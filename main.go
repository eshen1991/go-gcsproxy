package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	rawLog "log"
	"os"

	"github.com/byronwhitlock-google/go-mitmproxy/addon"
	"github.com/byronwhitlock-google/go-mitmproxy/proxy"
	"github.com/byronwhitlock-google/go-mitmproxy/web"
	log "github.com/sirupsen/logrus"
)

// makefile will turn this into a version
var Version = ".3"

type Config struct {
	version bool // show version

	Addr        string // proxy listen addr
	WebAddr     string // web interface listen addr
	SslInsecure bool   // not verify upstream server SSL/TLS certificates.

	CertPath string // path of generate cert files
	Debug    int    // debug mode: 1 - print debug log, 2 - show debug from

	Dump      string // dump filename
	DumpLevel int    // dump level: 0 - header, 1 - header + body

	// kms options
	KmsBucketKeyMapping string

	Upstream     string // upstream proxy
	UpstreamCert bool   // Connect to upstream server to look up certificate details. Default: True
}

// global config variable
var config *Config

func main() {
	config = loadConfig()
	if config.version {
		log.Infof("go-gcsproxy: %v", Version)
		Usage()
		os.Exit(0)
	}

	if config.Debug > 0 {
		rawLog.SetFlags(rawLog.LstdFlags | rawLog.Lshortfile)
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	if config.Debug == 2 {
		log.SetReportCaller(true)
	}
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	if config.KmsBucketKeyMapping == "" {
		log.Infof("\n>>> Please provide KMS Bucket Map.")
		os.Exit(0)

	} else {
		err := CheckKmsBucketKeyMapping()
		if err != nil {
			log.Infof("\n>>> unable to initialize KmsBucketKeyMapping. %v", err)
			os.Exit(0)
		}
	}

	opts := &proxy.Options{
		Debug:             config.Debug,
		Addr:              config.Addr,
		StreamLargeBodies: 1024 * 1024 * 1024 * 64, // TODO: we need to implement streaming intercept functions set to 64GB for now!
		SslInsecure:       config.SslInsecure,
		CaRootPath:        config.CertPath,
		Upstream:          config.Upstream,
	}

	p, err := proxy.NewProxy(opts)
	if err != nil {
		log.Fatal(err)
	}

	if !config.UpstreamCert {
		p.AddAddon(proxy.NewUpstreamCertAddon(false))
		log.Infoln("UpstreamCert config false")
	}

	p.AddAddon(&proxy.LogAddon{})
	p.AddAddon(web.NewWebAddon(config.WebAddr))

	p.AddAddon(&EncryptGcsPayload{})
	p.AddAddon(&DecryptGcsPayload{})
	p.AddAddon(&GetReqHeader{})

	if config.Dump != "" {
		dumper := addon.NewDumperWithFilename(config.Dump, config.DumpLevel)
		p.AddAddon(dumper)
	}

	configJson, _ := json.MarshalIndent(config, "", "\t")
	log.Infof("go-gcsproxy version '%v' Started. %v", config.version, string(configJson))
	log.Infof("Encryption enabled: %t", !IsEncryptDisabled())

	log.Fatal(p.Start())
}

func loadConfig() *Config {
	config := new(Config)

	defaultSslInsecure := envConfigBoolWithDefault("SSL_INSECURE", true)
	defaultCertPath := envConfigStringWithDefault("PROXY_CERT_PATH", "/proxy/certs")
	defaultDebug := envConfigIntWithDefault("DEBUG_LEVEL", 0)
	defaultKmsBucketKeyMapping := envConfigStringWithDefault("GCP_KMS_BUCKET_KEY_MAPPING", "")

	flag.BoolVar(&config.version, "version", false, "show go-gcsproxy version")
	flag.StringVar(&config.Addr, "port", ":9080", "proxy listen addr")
	flag.StringVar(&config.WebAddr, "web_port", ":9081", "web interface listen addr")
	flag.BoolVar(&config.SslInsecure, "ssl_insecure", defaultSslInsecure, "don't verify upstream server SSL/TLS certificates.")

	flag.StringVar(&config.CertPath, "cert_path", defaultCertPath, "path to cert. if 'mitmproxy-ca.pem' is not present here, it will be generated.")
	flag.IntVar(&config.Debug, "debug", defaultDebug, "debug level: 0 - ERROR, 1 - DEBUG, 2 - TRACE")
	flag.StringVar(&config.Dump, "dump", "", "filename to dump req/responses for debugging")
	flag.IntVar(&config.DumpLevel, "dump_level", 0, "dump level: 0 - header, 1 - header + body")
	flag.StringVar(&config.Upstream, "upstream", "", "upstream proxy")
	// "*:global-key" or "bucket/path:project/key,bucket2:key2" but the global key overrides all the other keys
	flag.StringVar(&config.KmsBucketKeyMapping, "kms_bucket_key_mappings", defaultKmsBucketKeyMapping, "Its the bucket name to KMS key map, payload will be encrypted with the bucket to key stored in KMS. KMS key should be in the format: projects/<project_id>/locations/<global|region>/keyRings/<key_ring>/cryptoKeys/<key>")

	flag.BoolVar(&config.UpstreamCert, "upstream_cert", false, "connect to upstream server to look up certificate details")
	flag.Parse()

	return config
}
func Usage() {
	flag.Usage()
	log.Info("\nEnvironment variables supported:")
	log.Info("  PROXY_CERT_PATH")
	log.Info("  SSL_INSECURE")
	log.Info("  DEBUG_LEVEL")
	log.Info("  GCP_KMS_BUCKET_KEY_MAPPING")
}

func CheckKmsBucketKeyMapping() error {
	var ctx = context.TODO()
	bucketKeyMap := bucketKeyMappings(config.KmsBucketKeyMapping)
	if bucketKeyMap == nil {
		return fmt.Errorf("No KmsBucketKeyMapping found")
	}
	for _, value := range bucketKeyMap {
		_, err := EncryptBytes(ctx, value, []byte("Hello, World!"))
		if err != nil {
			return err
		}
	}
	return nil
}
