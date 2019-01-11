syslog
======

fork of the standard go syslog package

This adds the ability to write to syslog daemons using TLS, as well as implementing this for Windows

```go

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/gabriel-samfira/syslog"
)

const caPem = `-----BEGIN CERTIFICATE-----
MIICXzCCAcqgAwIBAgIBADALBgkqhkiG9w0BAQUwRTENMAsGA1UEChMEanVqdTE0
MDIGA1UEAwwranVqdS1nZW5lcmF0ZWQgQ0EgZm9yIGVudmlyb25tZW50ICJyc3lz
bG9nIjAeFw0xNDA4MDUxMjEzNTBaFw0yNDA4MDUxMjE4NTBaMEUxDTALBgNVBAoT
BGp1anUxNDAyBgNVBAMMK2p1anUtZ2VuZXJhdGVkIENBIGZvciBlbnZpcm9ubWVu
dCAicnN5c2xvZyIwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBALSz4DWGHrXW
xp6uwwJ3j6amUhQajtGetkrPWLXp85gpdnwDgXgCOm/RXWHV2F2FtiSXkAf9FOQR
AOz2UhElHRMsv4+dsLJL9HfG2VtD6p73qR4vpwMYfIYb9ofHoK9A9tSpUoZRwZRz
wgoiayjeXvXMh9WRiszjln9dpYsUmZQlAgMBAAGjYzBhMA4GA1UdDwEB/wQEAwIA
pDAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBRtRlWT4zNaljsAYuaJo4epOwaH
HTAfBgNVHSMEGDAWgBRtRlWT4zNaljsAYuaJo4epOwaHHTALBgkqhkiG9w0BAQUD
gYEAAwi3/RUlgxt5xEQW3V4kgZmyAMrGt6uM417htZw/7E9CkfCFPjYKIITQKjAO
2ytOpL9dkJcDPW488vWkTBBqBSJWX6Vjz+T1Z6sebw24+VvvTo7oaQGhlJD4stLY
byTiSrVQmhaH5QPCErgdeBn6AZkIZ1XuB5VMoYTYbBLObO0=
-----END CERTIFICATE-----`

const cert = `-----BEGIN CERTIFICATE-----
MIICOTCCAaSgAwIBAgIBADALBgkqhkiG9w0BAQUwRTENMAsGA1UEChMEanVqdTE0
MDIGA1UEAwwranVqdS1nZW5lcmF0ZWQgQ0EgZm9yIGVudmlyb25tZW50ICJyc3lz
bG9nIjAeFw0xNDA4MDUxMjEzNTBaFw0yNDA4MDUxMjE4NTBaMBsxDTALBgNVBAoT
BGp1anUxCjAIBgNVBAMTASowgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBAOBc
CBEBj2K6dcV3xm1vqByyhki8dUl4AxmnrVDwr7pNKvgyf3t0qoY6/8P+/fphge8M
yFNS0cDmIL27PvUxFOdsPLFDEBeuY373L8EerYMq3Gp/M/UW4k/lwZEuRTKQ4oZ1
mvjXySKEAqroQ8Fq7wOLRkBORLbBFJ47au9U4HKhAgMBAAGjZzBlMA4GA1UdDwEB
/wQEAwIAqDATBgNVHSUEDDAKBggrBgEFBQcDATAdBgNVHQ4EFgQU8RsHN12K62sV
irTv3dPEFrVjV0swHwYDVR0jBBgwFoAUbUZVk+MzWpY7AGLmiaOHqTsGhx0wCwYJ
KoZIhvcNAQEFA4GBAKdb7/YA3u7SuGxXMEoFz6zqe51E+CfNhhToNXEHFX2JYRUk
aDvUNHDelSsclipo8LEBwvffcN9PH3ruWVlNusGyLjMFaKcuhjJHwv+AoOHpJgBd
AFWciBspXneItQs1wi5kwyFPphLJifEOS83Sc4jtqHj5lq8vjoYBzDLgrnHw
-----END CERTIFICATE-----`

const key = `-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDgXAgRAY9iunXFd8Ztb6gcsoZIvHVJeAMZp61Q8K+6TSr4Mn97
dKqGOv/D/v36YYHvDMhTUtHA5iC9uz71MRTnbDyxQxAXrmN+9y/BHq2DKtxqfzP1
FuJP5cGRLkUykOKGdZr418kihAKq6EPBau8Di0ZATkS2wRSeO2rvVOByoQIDAQAB
AoGAD/hdFqDOzQ9KvNCmzjlpdQl8J4dKrf0d82CNJLrNN2ywx1QI4QfP75gZhqEL
ARyZvCNjyxKVHa8D252NgLSKsUBTGllB3Dn9M8MZ9i9w6AapSwTwy9hxCrgB6ILC
6BnWW+HpuWq6v1Ft+lNycwoDwlevlpX7jfpmQTaNxYFg2jECQQDs354qlZs/Boqz
RTdgkM31kglcXUo8W4ZxU35DiVWsGb24boo6HurTwyqJBOogxDnWIZw4kgCbdRUW
FMA/04TtAkEA8nm8+WghdSgRDxXD486zzhrRnt6++vcARiJs4Mc621H9yjNwLrHz
2eIdWeE/2/xXtETWtGTX9ByQ8ufg3+kCBQJADDlF+kCaMFhwE+xAfVU7q66LmR6f
VBoNCBAc9fNCXo09gyUBMRqjV6Y8rbF5O5OkwG4fl7PBIEScf/U2LpUFyQJBAIdt
rzquCmHhKwX95hdKz+qB2CqfxpNted2yRJWXMSxmMxXIfRPXmJdNT49v27cGzgWF
nVXMLUHO4raJBHSLM/ECQQCAAuxb/GLAPDH9cbHo1BglU2mSzT81hSqanXcAapeh
2Y4xinXaXKxrgDFmPQJJZ2P+iCQuZp522N1+uro1zDlL
-----END RSA PRIVATE KEY-----`

func main() {
	caCert := x509.NewCertPool()
	ok := caCert.AppendCertsFromPEM([]byte(caPem))
	if !ok {
		fmt.Println("failed to parse root certificate")
		return
	}
	keyPair, err := tls.X509KeyPair([]byte(cert), []byte(key))
	if err != nil {
		fmt.Println("invalid keypair")
		return
	}

	tlsCfg := &tls.Config{
		ClientCAs: caCert,
		Certificates: []tls.Certificate{
			keyPair,
		},
		InsecureSkipVerify: true,
	}
	sLog, err := syslog.Dial("tcp", "192.168.200.51:6514", syslog.LOG_CRIT, "juju-syslog_test", tlsCfg)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer sLog.Close()
	err = sLog.Warning("hello")
	if err != nil {
		fmt.Println(err)
		return
	}
}
```
