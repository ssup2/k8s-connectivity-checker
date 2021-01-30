package kcchecker

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/ssup2/kcchecker/pkg/ip"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Const
const (
	ENV_NODE_NAME   = "OPT_NODE_NAME"
	ENV_INTERVAL_MS = "OPT_INTERVAL_MS"

	ENV_CHECK_PODPOD     = "OPT_CHECK_PODPOD"
	ENV_CHECK_PODSERVICE = "OPT_CHECK_PODSERVICE"
	ENV_CHECK_PODEX_ICMP = "OPT_CHECK_PODEX_ICMP"
	ENV_CHECK_PODEX_CONN = "OPT_CHECK_PODEX_CONN"
)

// Cmd
func New() (*cobra.Command, error) {
	var err error

	// Get and set options from env
	options := &Options{}
	options.nodeName = getEnv(ENV_NODE_NAME, "")
	if options.nodeName == "" {
		log.Error().Msg("Node name isn't set")
		return nil, fmt.Errorf("no node name")
	}
	options.intervalMS, err = strconv.Atoi(getEnv(ENV_INTERVAL_MS, "5000"))
	if err != nil {
		log.Error().Err(err).Msg("wrong intervalMS")
		return nil, err
	}

	options.checkPodPod, err = strconv.ParseBool(getEnv(ENV_CHECK_PODPOD, "true"))
	if err != nil {
		log.Error().Err(err).Msg("wrong check pod-pod option")
		return nil, err
	}
	options.checkPodService, err = strconv.ParseBool(getEnv(ENV_CHECK_PODSERVICE, "true"))
	if err != nil {
		log.Error().Err(err).Msg("wrong check pod-service option")
		return nil, err
	}
	options.CheckPodExICMP = getEnv(ENV_CHECK_PODEX_ICMP, "")
	if options.CheckPodExICMP != "" {
		for _, icmpIP := range strings.Split(options.CheckPodExICMP, ",") {
			if !ip.IsValidIP(icmpIP) {
				return nil, fmt.Errorf("wrong external ICMP IP format:%s", icmpIP)
			}
		}
	}
	options.CheckPodExConn = getEnv(ENV_CHECK_PODEX_CONN, "")
	if options.CheckPodExConn != "" {
		for _, connIPPort := range strings.Split(options.CheckPodExConn, ",") {
			if !ip.IsValidIPPort(connIPPort) {
				return nil, fmt.Errorf("wrong external connection IP/port format:%s", connIPPort)
			}
		}
	}

	// Check options
	if !options.checkPodPod && !options.checkPodService {
		log.Error().Msg("wrong check Pod-Service option")
		return nil, fmt.Errorf("all check disabled")
	}

	// Set command
	cmd := &cobra.Command{
		Use:                   "kcchecker",
		DisableFlagsInUseLine: true,
		Short:                 "check connectivity in a k8s cluster",
		Long:                  "check connectivity in a k8s cluster",
		Run: func(cmd *cobra.Command, args []string) {
			options.Run()
		},
	}

	return cmd, nil
}

// Options
type Options struct {
	nodeName   string
	intervalMS int

	checkPodPod     bool
	checkPodService bool
	CheckPodExICMP  string
	CheckPodExConn  string
}

func (o *Options) Run() {
	// Print options
	log.Info().Msgf("Applied options - %+v", *o)

	// Init k8s client
	log.Info().Msg("Init k8s client")
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Panic().Err(err).Msg("Failed to get k8s cluster config")
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to init k8s client")
	}

	// Get node's container runtime
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), o.nodeName, metav1.GetOptions{})
	nURL, err := url.Parse(node.Status.NodeInfo.ContainerRuntimeVersion)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to get node's container runtime")
	}
	nContRuntime := nURL.Scheme
	log.Info().Str("nodeContRuntime", nContRuntime).Msg("node's container runtime")

	// Check connectivity
	log.Info().Msg("Run kcchecker")
	for {
		// Get pod's and services' infos
		nPods, err := clientset.CoreV1().Pods("").
			List(context.TODO(), metav1.ListOptions{FieldSelector: "spec.nodeName=" + o.nodeName})
		if err != nil {
			panic(err.Error())
		}
		cPods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		services, err := clientset.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		// Eliminate duplicated pods IPs due to pods using host network namespaces
		nPodIPIDMap := make(map[string][]string)
		nPodIPNameMap := make(map[string][]string)
		for _, nPod := range nPods.Items {
			nPodID, err := getPodID(&nPod)
			if err != nil {
				log.Error().Err(err).Msg("Failed to get pod's id")
				continue
			}
			nPodName := nPod.Name
			nPodIP := nPod.Status.PodIP

			nPodIPIDMap[nPodIP] = append(nPodIPIDMap[nPodIP], nPodID)
			nPodIPNameMap[nPodIP] = append(nPodIPNameMap[nPodIP], nPodName)
		}

		cPodIPNameMap := make(map[string][]string)
		for _, cPod := range cPods.Items {
			cPodIP := cPod.Status.PodIP
			cPodName := cPod.Name

			cPodIPNameMap[cPodIP] = append(cPodIPNameMap[cPodIP], cPodName)
		}

		// Check Pod-Pod connectivity
		if o.checkPodPod {
			for nPodIP, nPodNames := range nPodIPNameMap {
				for cPodIP, cPodNames := range cPodIPNameMap {
					// Set logger
					logger := log.Info().
						Strs("srcPods", nPodNames).Str("srcIP", nPodIP).
						Strs("dstPods", cPodNames).Str("dstIP", cPodIP)

					// Run ping cmd in pod via cnsenter
					latencyMS, err := cmdPing(nContRuntime, nPodIPIDMap[nPodIP][0], cPodIP)
					if err != nil {
						logger.Err(err).Msg("Pod-Pod Error")
						continue
					}
					logger.Float64("latencyMS", latencyMS).Msg("Pod-Pod Ok")
				}
			}
		}

		// Check Pod-Service connectivity
		if o.checkPodService {
			for nPodIP, nPodNames := range nPodIPNameMap {
				for _, service := range services.Items {
					for _, servicePort := range service.Spec.Ports {
						// Init vars
						serviceIP := service.Spec.ClusterIP
						servicePort := servicePort.Port

						// Set logger
						logger := log.Info().
							Strs("srcPods", nPodNames).Str("srcIP", nPodIP).
							Str("dstServiceName", service.Name).Str("dstServiceIP", serviceIP).
							Int32("dstServicePort", servicePort)

						// Run ncat cmd in pod via cnsenter
						latencyMS, err := cmdNcatConn(nContRuntime, nPodIPIDMap[nPodIP][0], serviceIP, servicePort)
						if err != nil {
							logger.Err(err).Msg("Pod-Service Error")
							continue
						}
						logger.Float64("latencyMS", latencyMS).Msg("Pod-Service Ok")
					}
				}
			}
		}

		// Check Pod-External connectivity through ICMP
		if o.CheckPodExICMP != "" {
			for nPodIP, nPodNames := range nPodIPNameMap {
				for _, icmpIP := range strings.Split(o.CheckPodExICMP, ",") {
					// Set logger
					logger := log.Info().
						Strs("srcPods", nPodNames).Str("srcIP", nPodIP).
						Str("exICMPIP", icmpIP)

					// Run ping cmd in pod via cnsenter
					latencyMS, err := cmdPing(nContRuntime, nPodIPIDMap[nPodIP][0], icmpIP)
					if err != nil {
						logger.Err(err).Msg("Pod-External ICMP Error")
						continue
					}
					logger.Float64("latencyMS", latencyMS).Msg("Pod-External ICMP Ok")
				}
			}
		}

		// Check Pod-External connectivity through TCP/UDP
		if o.CheckPodExConn != "" {
			for nPodIP, nPodNames := range nPodIPNameMap {
				for _, connIPPort := range strings.Split(o.CheckPodExConn, ",") {
					// Init vars
					exIP, exPort, _ := ip.GetIPPort(connIPPort)

					// Set logger
					logger := log.Info().
						Strs("srcPods", nPodNames).Str("srcIP", nPodIP).
						Str("exConnIP", exIP).Int32("exConnPort", exPort)

					// Run ncat cmd in pod via cnsenter
					latencyMS, err := cmdNcatConn(nContRuntime, nPodIPIDMap[nPodIP][0], exIP, exPort)
					if err != nil {
						logger.Err(err).Msg("Pod-External Connection Error")
						continue
					}
					logger.Float64("latencyMS", latencyMS).Msg("Pod-External Connection Ok")
				}
			}
		}

		// Sleep
		time.Sleep(time.Duration(o.intervalMS) * time.Millisecond)
	}
}

// Helpers
func getEnv(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	return value
}

func getPodID(pod *corev1.Pod) (string, error) {
	// Get first container's ID and runtime
	// Because all container share net namespaces, Only get first container's ID to exec cnsenter
	if len(pod.Status.ContainerStatuses) <= 0 {
		return "", fmt.Errorf("no container info")
	}
	u, err := url.Parse(pod.Status.ContainerStatuses[0].ContainerID)
	if err != nil {
		return "", fmt.Errorf("parse first container's ID error")
	}
	return u.Host, nil
}
