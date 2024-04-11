package kubernetes

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestNewCluster_SingleNodeRKE2_MissingConfig(t *testing.T) {
	kubernetes := &image.Kubernetes{
		Version: "v1.29.0+rke2r1",
		Network: image.Network{
			APIHost: "api.suse.edge.com",
			APIVIP:  "192.168.122.50",
		},
	}

	cluster, err := NewCluster(kubernetes, "")
	require.NoError(t, err)

	require.NotNil(t, cluster.ServerConfig)
	assert.Equal(t, "cilium", cluster.ServerConfig["cni"])
	assert.Equal(t, []string{"192.168.122.50", "api.suse.edge.com"}, cluster.ServerConfig["tls-san"])
	assert.Nil(t, cluster.ServerConfig["token"])
	assert.Nil(t, cluster.ServerConfig["server"])
	assert.Nil(t, cluster.ServerConfig["selinux"])
	assert.Nil(t, cluster.ServerConfig["disable"])

	assert.Empty(t, cluster.InitialiserName)
	assert.Nil(t, cluster.InitialiserConfig)
	assert.Nil(t, cluster.AgentConfig)
}

func TestNewCluster_SingleNodeK3s_MissingConfig(t *testing.T) {
	kubernetes := &image.Kubernetes{
		Version: "v1.29.0+k3s1",
		Network: image.Network{
			APIHost: "api.suse.edge.com",
			APIVIP:  "192.168.122.50",
		},
	}

	cluster, err := NewCluster(kubernetes, "")
	require.NoError(t, err)

	require.NotNil(t, cluster.ServerConfig)
	assert.Equal(t, []string{"192.168.122.50", "api.suse.edge.com"}, cluster.ServerConfig["tls-san"])
	assert.Equal(t, []string{"servicelb"}, cluster.ServerConfig["disable"])
	assert.Nil(t, cluster.ServerConfig["cni"])
	assert.Nil(t, cluster.ServerConfig["token"])
	assert.Nil(t, cluster.ServerConfig["server"])
	assert.Nil(t, cluster.ServerConfig["selinux"])

	assert.Empty(t, cluster.InitialiserName)
	assert.Nil(t, cluster.InitialiserConfig)
	assert.Nil(t, cluster.AgentConfig)
}

func TestNewCluster_SingleNode_ExistingConfig(t *testing.T) {
	kubernetes := &image.Kubernetes{
		Network: image.Network{
			APIHost: "api.suse.edge.com",
			APIVIP:  "192.168.122.50",
		},
	}

	cluster, err := NewCluster(kubernetes, "testdata")
	require.NoError(t, err)

	require.NotNil(t, cluster.ServerConfig)
	assert.Equal(t, "calico", cluster.ServerConfig["cni"])
	assert.Equal(t, "totally-not-generated-one", cluster.ServerConfig["token"])
	assert.Equal(t, []string{"192.168.122.50", "api.suse.edge.com"}, cluster.ServerConfig["tls-san"])
	assert.Equal(t, true, cluster.ServerConfig["selinux"])
	assert.Nil(t, cluster.ServerConfig["server"])

	assert.Empty(t, cluster.InitialiserName)
	assert.Nil(t, cluster.InitialiserConfig)
	assert.Nil(t, cluster.AgentConfig)
}

func TestNewCluster_MultiNodeRKE2_MissingConfig(t *testing.T) {
	kubernetes := &image.Kubernetes{
		Version: "v1.29.0+rke2r1",
		Network: image.Network{
			APIHost: "api.suse.edge.com",
			APIVIP:  "192.168.122.50",
		},
		Nodes: []image.Node{
			{
				Hostname: "node1.suse.com",
				Type:     "server",
			},
			{
				Hostname: "node2.suse.com",
				Type:     "agent",
			},
		},
	}

	cluster, err := NewCluster(kubernetes, "")
	require.NoError(t, err)

	assert.Equal(t, "node1.suse.com", cluster.InitialiserName)

	require.NotNil(t, cluster.InitialiserConfig)
	clusterToken := cluster.InitialiserConfig["token"]
	require.NotPanics(t, func() {
		uuid.MustParse(clusterToken.(string))
	})
	assert.Equal(t, "cilium", cluster.InitialiserConfig["cni"])
	assert.Equal(t, []string{"192.168.122.50", "api.suse.edge.com"}, cluster.InitialiserConfig["tls-san"])
	assert.Equal(t, false, cluster.InitialiserConfig["selinux"])
	assert.Nil(t, cluster.InitialiserConfig["server"])

	require.NotNil(t, cluster.ServerConfig)
	assert.Equal(t, "cilium", cluster.ServerConfig["cni"])
	assert.Equal(t, []string{"192.168.122.50", "api.suse.edge.com"}, cluster.ServerConfig["tls-san"])
	assert.Equal(t, clusterToken, cluster.ServerConfig["token"])
	assert.Equal(t, "https://192.168.122.50:9345", cluster.ServerConfig["server"])
	assert.Equal(t, false, cluster.ServerConfig["selinux"])

	require.NotNil(t, cluster.AgentConfig)
	assert.Equal(t, "cilium", cluster.AgentConfig["cni"])
	assert.Equal(t, clusterToken, cluster.AgentConfig["token"])
	assert.Equal(t, "https://192.168.122.50:9345", cluster.AgentConfig["server"])
	assert.Equal(t, false, cluster.AgentConfig["selinux"])
	assert.Nil(t, cluster.AgentConfig["tls-san"])
	assert.Nil(t, cluster.AgentConfig["debug"])
}

func TestNewCluster_MultiNodeRKE2_ExistingConfig(t *testing.T) {
	kubernetes := &image.Kubernetes{
		Version: "v1.29.0+rke2r1",
		Network: image.Network{
			APIHost: "api.suse.edge.com",
			APIVIP:  "192.168.122.50",
		},
		Nodes: []image.Node{
			{
				Hostname: "node1.suse.com",
				Type:     "server",
			},
			{
				Hostname: "node2.suse.com",
				Type:     "agent",
			},
		},
	}

	cluster, err := NewCluster(kubernetes, "testdata")
	require.NoError(t, err)

	assert.Equal(t, "node1.suse.com", cluster.InitialiserName)

	require.NotNil(t, cluster.InitialiserConfig)
	assert.Equal(t, "calico", cluster.InitialiserConfig["cni"])
	assert.Equal(t, []string{"192.168.122.50", "api.suse.edge.com"}, cluster.InitialiserConfig["tls-san"])
	assert.Equal(t, "totally-not-generated-one", cluster.InitialiserConfig["token"])
	assert.Nil(t, cluster.InitialiserConfig["server"])
	assert.Equal(t, true, cluster.InitialiserConfig["selinux"])
	assert.Nil(t, cluster.InitialiserConfig["debug"])

	require.NotNil(t, cluster.ServerConfig)
	assert.Equal(t, "calico", cluster.ServerConfig["cni"])
	assert.Equal(t, []string{"192.168.122.50", "api.suse.edge.com"}, cluster.ServerConfig["tls-san"])
	assert.Equal(t, "totally-not-generated-one", cluster.ServerConfig["token"])
	assert.Equal(t, "https://192.168.122.50:9345", cluster.ServerConfig["server"])
	assert.Equal(t, true, cluster.ServerConfig["selinux"])
	assert.Nil(t, cluster.ServerConfig["debug"])

	require.NotNil(t, cluster.AgentConfig)
	assert.Equal(t, "calico", cluster.AgentConfig["cni"])
	assert.Equal(t, "totally-not-generated-one", cluster.AgentConfig["token"])
	assert.Equal(t, "https://192.168.122.50:9345", cluster.AgentConfig["server"])
	assert.Equal(t, true, cluster.AgentConfig["debug"])
	assert.Equal(t, true, cluster.AgentConfig["selinux"])
	assert.Nil(t, cluster.AgentConfig["tls-san"])
}

func TestNewCluster_MultiNode_MissingInitialiser(t *testing.T) {
	kubernetes := &image.Kubernetes{
		Nodes: []image.Node{
			{
				Hostname: "node1.suse.com",
				Type:     "agent",
			},
			{
				Hostname: "node2.suse.com",
				Type:     "agent",
			},
		},
	}

	cluster, err := NewCluster(kubernetes, "")

	assert.Error(t, err, "failed to determine cluster initialiser")
	assert.Nil(t, cluster)
}

func TestIdentifyInitialiserNode(t *testing.T) {
	tests := []struct {
		name         string
		nodes        []image.Node
		expectedNode string
	}{
		{
			name:         "Empty list of nodes",
			expectedNode: "",
		},
		{
			name: "Agent list",
			nodes: []image.Node{
				{
					Hostname: "host1",
					Type:     "agent",
				},
				{
					Hostname: "host2",
					Type:     "agent",
				},
			},

			expectedNode: "",
		},
		{
			name: "Server node labeled as initialiser",
			nodes: []image.Node{
				{
					Hostname: "host1",
					Type:     "agent",
				},
				{
					Hostname: "host2",
					Type:     "server",
				},
				{
					Hostname:    "host3",
					Type:        "server",
					Initialiser: true,
				},
			},
			expectedNode: "host3",
		},
		{
			name: "Initialiser as first server node in list",
			nodes: []image.Node{
				{
					Hostname: "host1",
					Type:     "agent",
				},
				{
					Hostname: "host2",
					Type:     "server",
				},
				{
					Hostname: "host3",
					Type:     "server",
				},
			},
			expectedNode: "host2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubernetes := &image.Kubernetes{Nodes: test.nodes}
			assert.Equal(t, test.expectedNode, identifyInitialiserNode(kubernetes))
		})
	}
}

func TestSetClusterAPIAddress(t *testing.T) {
	config := map[string]any{}

	setClusterAPIAddress(config, "", 9345)
	assert.NotContains(t, config, "server")

	setClusterAPIAddress(config, "192.168.122.50", 9345)
	assert.Equal(t, "https://192.168.122.50:9345", config["server"])
}

func TestAppendClusterTLSSAN(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]any
		apiHost        string
		expectedTLSSAN any
	}{
		{
			name:           "Empty TLS SAN",
			config:         map[string]any{},
			apiHost:        "",
			expectedTLSSAN: nil,
		},
		{
			name:           "Missing TLS SAN",
			config:         map[string]any{},
			apiHost:        "api.cluster01.hosted.on.edge.suse.com",
			expectedTLSSAN: []string{"api.cluster01.hosted.on.edge.suse.com"},
		},
		{
			name: "Invalid TLS SAN",
			config: map[string]any{
				"tls-san": 5,
			},
			apiHost:        "api.cluster01.hosted.on.edge.suse.com",
			expectedTLSSAN: []string{"api.cluster01.hosted.on.edge.suse.com"},
		},
		{
			name: "Existing TLS SAN string",
			config: map[string]any{
				"tls-san": "api.edge1.com, api.edge2.com",
			},
			apiHost:        "api.cluster01.hosted.on.edge.suse.com",
			expectedTLSSAN: []string{"api.edge1.com", "api.edge2.com", "api.cluster01.hosted.on.edge.suse.com"},
		},
		{
			name: "Existing TLS SAN string list",
			config: map[string]any{
				"tls-san": []string{"api.edge1.com", "api.edge2.com"},
			},
			apiHost:        "api.cluster01.hosted.on.edge.suse.com",
			expectedTLSSAN: []string{"api.edge1.com", "api.edge2.com", "api.cluster01.hosted.on.edge.suse.com"},
		},
		{
			name: "Existing TLS SAN list",
			config: map[string]any{
				"tls-san": []any{"api.edge1.com", "api.edge2.com"},
			},
			apiHost:        "api.cluster01.hosted.on.edge.suse.com",
			expectedTLSSAN: []any{"api.edge1.com", "api.edge2.com", "api.cluster01.hosted.on.edge.suse.com"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			appendClusterTLSSAN(test.config, test.apiHost)
			assert.Equal(t, test.expectedTLSSAN, test.config["tls-san"])
		})
	}
}

func TestAppendClusterDisabledServices(t *testing.T) {
	tests := []struct {
		name             string
		config           map[string]any
		service          string
		expectedServices any
	}{
		{
			name:             "Empty services",
			config:           map[string]any{},
			service:          "",
			expectedServices: nil,
		},
		{
			name:             "Missing service",
			config:           map[string]any{},
			service:          "servicelb",
			expectedServices: []string{"servicelb"},
		},
		{
			name: "Invalid service",
			config: map[string]any{
				"disable": 5,
			},
			service:          "servicelb",
			expectedServices: []string{"servicelb"},
		},
		{
			name: "Existing service string",
			config: map[string]any{
				"disable": "traefik",
			},
			service:          "servicelb",
			expectedServices: []string{"traefik", "servicelb"},
		},
		{
			name: "Existing service string list",
			config: map[string]any{
				"disable": []string{"traefik"},
			},
			service:          "servicelb",
			expectedServices: []string{"traefik", "servicelb"},
		},
		{
			name: "Existing service list",
			config: map[string]any{
				"disable": []any{"traefik"},
			},
			service:          "servicelb",
			expectedServices: []any{"traefik", "servicelb"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			appendDisabledServices(test.config, test.service)
			assert.Equal(t, test.expectedServices, test.config["disable"])
		})
	}
}

func TestServersCount(t *testing.T) {
	nodes := []image.Node{
		{
			Hostname: "node1",
			Type:     "server",
		},
		{
			Hostname: "node2",
			Type:     "agent",
		},
		{
			Hostname: "node3",
			Type:     "server",
		},
	}

	assert.Equal(t, 2, ServersCount(nodes))
	assert.Equal(t, 0, ServersCount([]image.Node{}))
}
