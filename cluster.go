package gocouchbase

import "time"
import "fmt"
import "github.com/couchbase/gocouchbaseio"
import "net/http"
import "crypto/tls"

type Cluster struct {
	spec              connSpec
	connectionTimeout time.Duration
}

func Connect(connSpecStr string) (*Cluster, error) {
	spec := parseConnSpec(connSpecStr)
	if spec.Scheme == "" {
		spec.Scheme = "http"
	}
	if spec.Scheme != "couchbase" && spec.Scheme != "couchbases" && spec.Scheme != "http" {
		panic("Unsupported Scheme!")
	}
	cluster := &Cluster{
		spec:              spec,
		connectionTimeout: 10000 * time.Millisecond,
	}
	return cluster, nil
}

func (c *Cluster) OpenBucket(bucket, password string) (*Bucket, error) {
	var memdHosts []string
	var httpHosts []string
	isHttpHosts := c.spec.Scheme == "http"
	isSslHosts := c.spec.Scheme == "couchbases"
	for _, specHost := range c.spec.Hosts {
		if specHost.Port == 0 {
			if !isHttpHosts {
				if !isSslHosts {
					specHost.Port = 11210
				} else {
					specHost.Port = 11207
				}
			} else {
				panic("HTTP configuration not yet supported")
				//specHost.Port = 8091
			}
		}
		memdHosts = append(memdHosts, fmt.Sprintf("%s:%d", specHost.Host, specHost.Port))
	}

	authFn := func(srv gocouchbaseio.MemdAuthClient) error {
		fmt.Printf("Want to auth for %s\n", srv.Address())

		// Build PLAIN auth data
		userBuf := []byte(bucket)
		passBuf := []byte(password)
		authData := make([]byte, 1+len(userBuf)+1+len(passBuf))
		authData[0] = 0
		copy(authData[1:], userBuf)
		authData[1+len(userBuf)] = 0
		copy(authData[1+len(userBuf)+1:], passBuf)

		// Execute PLAIN authentication
		authR, err := srv.SaslAuth([]byte("PLAIN"), authData)
		fmt.Printf("SaslAuth said %v, %s\n", err, authR)

		return nil
	}
	cli, err := gocouchbaseio.CreateAgent(memdHosts, httpHosts, isSslHosts, authFn)
	if err != nil {
		return nil, err
	}

	return &Bucket{
		client: cli,
		httpCli: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}, nil
}