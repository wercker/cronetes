package main

import (
	"os"

	"gopkg.in/urfave/cli.v1"

	log "github.com/Sirupsen/logrus"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

func main() {
	app := cli.NewApp()
	app.Name = "cronetes"
	app.Version = "1.0.0"
	app.Usage = "Simple cron daemon that creates kubernetes Jobs"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "Add extra debug log message",
			EnvVar: "CRONETES_DEBUG",
		},
		cli.BoolFlag{
			Name:   "kube-in-cluster",
			Usage:  "Use the kubernetes service account (other kubernetes config are ignored)",
			EnvVar: "KUBE_IN_CLUSTER",
		},
		cli.StringFlag{
			Name:   "kube-endpoint",
			Usage:  "The address and port of the Kubernetes API server",
			EnvVar: "KUBE_ENDPOINT",
		},
		cli.StringFlag{
			Name:   "kube-username",
			Usage:  "Username for the basic authentication to the API server",
			EnvVar: "KUBE_USERNAME",
		},
		cli.StringFlag{
			Name:   "kube-password",
			Usage:  "Password for basic authentication to the API server",
			EnvVar: "KUBE_PASSWORD",
		},
		cli.StringFlag{
			Name:   "kube-token",
			Usage:  "Bearer token for authentication to the API server",
			EnvVar: "KUBE_TOKEN",
		},
		cli.BoolFlag{
			Name:   "kube-insecure",
			Usage:  "If true, the server's certificate will not be checked for validaty",
			EnvVar: "KUBE_INSECURE",
		},
		cli.StringFlag{
			Name:   "kube-ca-file",
			Usage:  "Path to a cert. file for the certificate authority",
			EnvVar: "KUBE_CA_FILE",
		},
		cli.StringFlag{
			Name:   "kube-key-file",
			Usage:  "Path to a client key file for TLS",
			EnvVar: "KUBE_KEY_FILE",
		},
		cli.StringFlag{
			Name:   "kube-cert-file",
			Usage:  "Path to a client certificate file for TLS",
			EnvVar: "KUBE_CERT_FILE",
		},
		cli.StringFlag{
			Name:   "kube-namespace",
			Usage:  "If present, the namespace scope for this CLI request",
			Value:  api.NamespaceDefault,
			EnvVar: "KUBE_NAMESPACE",
		},
	}
	app.Commands = []cli.Command{
		cronCommand,
		reapCommand,
	}

	app.Run(os.Args)
}

func ParseGlobalOptions(c *cli.Context) *GlobalOptions {
	return &GlobalOptions{
		Debug:         c.GlobalBool("debug"),
		KubeCAFile:    c.GlobalString("kube-ca-file"),
		KubeCertFile:  c.GlobalString("kube-cert-file"),
		KubeEndpoint:  c.GlobalString("kube-endpoint"),
		KubeInCluster: c.GlobalBool("kube-in-cluster"),
		KubeInsecure:  c.GlobalBool("kube-insecure"),
		KubeKeyFile:   c.GlobalString("kube-key-file"),
		KubeNamespace: c.GlobalString("kube-namespace"),
		KubePassword:  c.GlobalString("kube-password"),
		KubeToken:     c.GlobalString("kube-token"),
		KubeUsername:  c.GlobalString("kube-username"),
	}
}

type GlobalOptions struct {
	Debug         bool
	KubeCAFile    string
	KubeCertFile  string
	KubeEndpoint  string
	KubeInCluster bool
	KubeInsecure  bool
	KubeKeyFile   string
	KubeNamespace string
	KubePassword  string
	KubeToken     string
	KubeUsername  string
}

func createClient(o *GlobalOptions) (*client.Client, error) {
	if o.KubeInCluster {
		return client.NewInCluster()
	}

	config := &restclient.Config{
		BearerToken: o.KubeToken,
		Host:        o.KubeEndpoint,
		Insecure:    o.KubeInsecure,
		Password:    o.KubePassword,
		Username:    o.KubeUsername,
	}

	caFile := o.KubeCAFile
	certFile := o.KubeCertFile
	keyFile := o.KubeKeyFile

	if caFile != "" && certFile != "" && keyFile != "" {
		config.TLSClientConfig = restclient.TLSClientConfig{
			CAFile:   caFile,
			CertFile: certFile,
			KeyFile:  keyFile,
		}
	}

	return client.New(config)
}

func setLogLevel(debug bool) {
	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}
