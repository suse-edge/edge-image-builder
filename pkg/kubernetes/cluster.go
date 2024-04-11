package kubernetes

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	serverConfigFile = "server.yaml"
	agentConfigFile  = "agent.yaml"

	tokenKey        = "token"
	cniKey          = "cni"
	cniDefaultValue = image.CNITypeCilium
	serverKey       = "server"
	tlsSANKey       = "tls-san"
	disableKey      = "disable"
	clusterInitKey  = "cluster-init"
	selinuxKey      = "selinux"
)

type Cluster struct {
	// InitialiserName is the hostname of the initialiser node.
	// Defaults to the first configured server if not explicitly selected.
	InitialiserName string
	// InitialiserConfig contains the server configuration for the node initialising a multi node cluster.
	InitialiserConfig map[string]any
	// ServerConfig contains the server configurations for a single node cluster
	// or the additional server nodes in a multi node cluster.
	ServerConfig map[string]any
	// AgentConfig contains the agent configurations in multi node clusters.
	AgentConfig map[string]any
}

func NewCluster(kubernetes *image.Kubernetes, configPath string) (*Cluster, error) {
	serverConfigPath := filepath.Join(configPath, serverConfigFile)
	serverConfig, err := ParseKubernetesConfig(serverConfigPath)
	if err != nil {
		return nil, fmt.Errorf("parsing server config: %w", err)
	}

	if len(kubernetes.Nodes) < 2 {
		setSingleNodeConfigDefaults(kubernetes, serverConfig)
		return &Cluster{ServerConfig: serverConfig}, nil
	}

	setMultiNodeConfigDefaults(kubernetes, serverConfig)

	agentConfigPath := filepath.Join(configPath, agentConfigFile)
	agentConfig, err := ParseKubernetesConfig(agentConfigPath)
	if err != nil {
		return nil, fmt.Errorf("parsing agent config: %w", err)
	}

	// Ensure the agent uses the same cluster configuration values as the server
	agentConfig[tokenKey] = serverConfig[tokenKey]
	agentConfig[serverKey] = serverConfig[serverKey]
	agentConfig[selinuxKey] = serverConfig[selinuxKey]
	if strings.Contains(kubernetes.Version, image.KubernetesDistroRKE2) {
		agentConfig[cniKey] = serverConfig[cniKey]
	}

	// Create the initialiser server config
	initialiserConfig := map[string]any{}
	for k, v := range serverConfig {
		initialiserConfig[k] = v
	}
	delete(initialiserConfig, serverKey)
	if strings.Contains(kubernetes.Version, image.KubernetesDistroK3S) {
		initialiserConfig[clusterInitKey] = true
	}

	initialiser := identifyInitialiserNode(kubernetes)
	if initialiser == "" {
		return nil, fmt.Errorf("failed to determine cluster initialiser")
	}

	return &Cluster{
		InitialiserName:   initialiser,
		InitialiserConfig: initialiserConfig,
		ServerConfig:      serverConfig,
		AgentConfig:       agentConfig,
	}, nil
}

func ParseKubernetesConfig(configFile string) (map[string]any, error) {
	config := map[string]any{}

	b, err := os.ReadFile(configFile)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("reading kubernetes config file '%s': %w", configFile, err)
		}

		zap.S().Warnf("Kubernetes config file '%s' was not provided", configFile)

		// Use an empty config which will be automatically populated later
		return config, nil
	}

	if err = yaml.Unmarshal(b, &config); err != nil {
		return nil, fmt.Errorf("parsing kubernetes config file '%s': %w", configFile, err)
	}

	return config, nil
}

func identifyInitialiserNode(kubernetes *image.Kubernetes) string {
	for _, node := range kubernetes.Nodes {
		if node.Initialiser {
			return node.Hostname
		}
	}

	// Use the first server node as an initialiser
	for _, node := range kubernetes.Nodes {
		if node.Type == image.KubernetesNodeTypeServer {
			zap.S().Infof("Using '%s' as the cluster initialiser, as one wasn't explicitly selected", node.Hostname)
			return node.Hostname
		}
	}

	return ""
}

func setSingleNodeConfigDefaults(kubernetes *image.Kubernetes, config map[string]any) {
	if strings.Contains(kubernetes.Version, image.KubernetesDistroRKE2) {
		setClusterCNI(config)
	}
	if kubernetes.Network.APIVIP != "" {
		appendClusterTLSSAN(config, kubernetes.Network.APIVIP)

		if strings.Contains(kubernetes.Version, image.KubernetesDistroK3S) {
			appendDisabledServices(config, "servicelb")
		}
	}
	if kubernetes.Network.APIHost != "" {
		appendClusterTLSSAN(config, kubernetes.Network.APIHost)
	}
	delete(config, serverKey)
}

func setMultiNodeConfigDefaults(kubernetes *image.Kubernetes, config map[string]any) {
	const (
		k3sServerPort  = 6443
		rke2ServerPort = 9345
	)

	if strings.Contains(kubernetes.Version, image.KubernetesDistroRKE2) {
		setClusterAPIAddress(config, kubernetes.Network.APIVIP, rke2ServerPort)
		setClusterCNI(config)
	} else {
		setClusterAPIAddress(config, kubernetes.Network.APIVIP, k3sServerPort)
		appendDisabledServices(config, "servicelb")
	}

	setClusterToken(config)
	appendClusterTLSSAN(config, kubernetes.Network.APIVIP)
	setSELinux(config)
	if kubernetes.Network.APIHost != "" {
		appendClusterTLSSAN(config, kubernetes.Network.APIHost)
	}
}

func setClusterToken(config map[string]any) {
	if _, ok := config[tokenKey].(string); ok {
		return
	}

	token := uuid.NewString()

	zap.S().Infof("Generated cluster token: %s", token)
	config[tokenKey] = token
}

func setClusterCNI(config map[string]any) {
	if _, ok := config[cniKey]; ok {
		return
	}

	auditMessage := fmt.Sprintf("The Kubernetes CNI is not explicitly set, defaulting to '%s'.", cniDefaultValue)
	log.Audit(auditMessage)

	zap.S().Infof("CNI not set in config file, proceeding with CNI: %s", cniDefaultValue)

	config[cniKey] = cniDefaultValue
}

func setClusterAPIAddress(config map[string]any, apiAddress string, port int) {
	if apiAddress == "" {
		zap.S().Warn("Attempted to set an empty cluster API address")
		return
	}

	config[serverKey] = fmt.Sprintf("https://%s:%d", apiAddress, port)
}

func setSELinux(config map[string]any) {
	if _, ok := config[selinuxKey].(bool); ok {
		return
	}

	config[selinuxKey] = false
}

func appendClusterTLSSAN(config map[string]any, address string) {
	if address == "" {
		zap.S().Warn("Attempted to append TLS SAN with an empty address")
		return
	}

	tlsSAN, ok := config[tlsSANKey]
	if !ok {
		config[tlsSANKey] = []string{address}
		return
	}

	switch v := tlsSAN.(type) {
	case string:
		var tlsSANs []string
		for _, san := range strings.Split(v, ",") {
			tlsSANs = append(tlsSANs, strings.TrimSpace(san))
		}
		tlsSANs = append(tlsSANs, address)
		config[tlsSANKey] = tlsSANs
	case []string:
		v = append(v, address)
		config[tlsSANKey] = v
	case []any:
		v = append(v, address)
		config[tlsSANKey] = v
	default:
		zap.S().Warnf("Ignoring invalid 'tls-san' value: %v", v)
		config[tlsSANKey] = []string{address}
	}
}

func appendDisabledServices(config map[string]any, service string) {
	if service == "" {
		zap.S().Warn("Attempted to disable an empty service")
		return
	}

	disabledServices, ok := config[disableKey]
	if !ok {
		config[disableKey] = []string{service}
		return
	}

	switch v := disabledServices.(type) {
	case string:
		var services []string
		for _, s := range strings.Split(v, ",") {
			services = append(services, strings.TrimSpace(s))
		}
		services = append(services, service)
		config[disableKey] = services
	case []string:
		v = append(v, service)
		config[disableKey] = v
	case []any:
		v = append(v, service)
		config[disableKey] = v
	default:
		zap.S().Warnf("Ignoring invalid 'disable' value: %v", v)
		config[disableKey] = []string{service}
	}
}

func ServersCount(nodes []image.Node) int {
	var servers int

	for _, node := range nodes {
		if node.Type == image.KubernetesNodeTypeServer {
			servers++
		}
	}

	return servers
}
