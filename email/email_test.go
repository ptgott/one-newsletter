package email

import "testing"

// Created a key/cert using this command. It's valid for 100 years from
// Jan 23, 2021.
// $ openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 36500 -nodes

const testCert string = `-----BEGIN CERTIFICATE-----
MIIFbTCCA1WgAwIBAgIUP7rk4l0GpkGgz7Wj3MvHvIauYvcwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAgFw0yMTAxMjMxNzAwNDNaGA8yMTIw
MTIzMDE3MDA0M1owRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUx
ITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDCCAiIwDQYJKoZIhvcN
AQEBBQADggIPADCCAgoCggIBAP3WOpAKIEM3X0c6UyXiirNn3YNU8GaYkIRgqzeK
KIbyxYbP7n0DvaEhxmnRsdqEHRzT/tXzX2pCnTxFilxrWpHW1yBdEDQu0IdakkKu
+rMkL6sII58t/FYnQcjtsn5+pZ3Gy5llItzxlEWKrt8rh4wZhaFBKWY1TSXUXAPC
45aDZe73QuZK7v2eZmKr6TlOR1LziAkgVIwYcujEDUtZV/vXyxVSd4+RRaoohlBc
vfUIpBWT7HOxA7ZO1A8cTuleU5JSjIKh5Guen/YlS57vca5xQ/lVQ7HZcxrZ4VJp
ZBwiA6wpQuEUquuENymUq0K6gZ4MiobCRfLfCLD8tvYsfhX+QSJVwkEVTyaiS7Yz
4PbmCAumqjdbp80x4koUdfXYFqbdvqK89Nmp4rvkhNR7i1WjOCtt/LbJq11jQdtq
BxKHhOWlZAPnZSJS6TlS3X6vUZTDSHkK6H6WGUHBwE+XbFJDK5w7y/LSwTN43xbi
YD/pujaKgf/2zVNm0UrP1W3Yc53wesfza7PjRYAIQ4KsuONjw6e+JQ4Rn+0uv1UL
NjHMVdl10YuaM2v2WHo1EMZLre7CW7WuiK3+UCfGG8mKt55UdumSv8VsZShmHXN+
p7p2Fv4SB0N6egbpOvA2gIX5nENAt1ZYQD+ebI252L8Jp9mAbFyycSlZ+aEzXEHh
eiqdAgMBAAGjUzBRMB0GA1UdDgQWBBT+c0WAwHwJo71UBugi6P69lsqcqjAfBgNV
HSMEGDAWgBT+c0WAwHwJo71UBugi6P69lsqcqjAPBgNVHRMBAf8EBTADAQH/MA0G
CSqGSIb3DQEBCwUAA4ICAQBu3lzPCZ7DvdN1GWsMAHBnwDMFY7kEubcHn+fWeQdc
T5CYu/NYmU3BzfK640cm9LIXCw21QICRm+sitA6PQNkeEE7spzj53EkL/Z7iJCGB
ZGFU/ib4SBIa7ZlujjfVlwv6kUitgpB45Z7k5Up9j7M0jUkciw/7h+kM2Oe4L83C
XkHulFipp3CTTJPVh3052B0JAAQF0SK7yglVn1xnTc/aP/5ub/sKPTpTpGKMuoPp
0u2rkes6qC6m7poilzvTnTDvzs8LifJyFVTykas2X33+7eMUF5Ze0f8AYMSSiBwR
m1NPTgE1aCdGVp8u4oNFVy2QUm2Av/dCx0eYpFZX8tP5wR1FoiN1t/+VS4NKa3WT
ZgdV80IEjMm3hwBZ1diM3pkLdPw8qFtsvtKu92oFU/rOyU3X0qF1mMCD1Wrar7Ph
lGFnCdRdosfbtyMcItJLXnjc6j++XWgy0BrsnXOPnfY605B8fgAAov9XR26qga7L
k3XoEZjm8NqfgJtCYS6HwcSH+IdCMrxZXT0B25UgM2ahA0TUmFMTQQpeyb7q14cc
Pyqbu+Q3Tj3uqlNrgpLdHl9r5pzm/vXzh/yizxT+4vYwWOuPZ58KP1agWtsiUwTJ
hIoIC+wvuQTMTMglWqDdvCvsvkWufNp/21Hw0r3gfzoxyIA0MEZR0yJPGQeFFtuH
fA==
-----END CERTIFICATE-----	
`

const testKey string = `
-----BEGIN PRIVATE KEY-----
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
G4PPQiy08jduVrW1dmlw8abbuDGki04=
-----END PRIVATE KEY-----
`

func TestNewSMTPClient(t *testing.T) {
	testCases := []struct {
		description      string
		shouldRaiseError bool
		userConfig       UserConfig
	}{
		{
			description:      "valid case",
			shouldRaiseError: false,
			userConfig: UserConfig{
				RelayAddress: "smtp://localhost:587",
				Key:          []byte(testKey),
				Cert:         []byte(testCert),
				Username:     "user1",
				Password:     "1234abcd",
				FromAddress:  "no-reply@example.com",
				ToAddress:    "me@example.com",
			},
		},
		{
			description:      "valid case with no url scheme",
			shouldRaiseError: false,
			userConfig: UserConfig{
				RelayAddress: "localhost:587",
				Key:          []byte(testKey),
				Cert:         []byte(testCert),
				Username:     "user1",
				Password:     "1234abcd",
				FromAddress:  "no-reply@example.com",
				ToAddress:    "me@example.com",
			},
		},
		{
			description:      "no address",
			shouldRaiseError: true,
			userConfig: UserConfig{
				Key:         []byte(testKey),
				Cert:        []byte(testCert),
				Username:    "user1",
				Password:    "1234abcd",
				FromAddress: "no-reply@example.com",
				ToAddress:   "me@example.com",
			},
		},
		{
			description:      "no key",
			shouldRaiseError: true,
			userConfig: UserConfig{
				RelayAddress: "smtp://localhost:587",
				Cert:         []byte(testCert),
				Username:     "user1",
				Password:     "1234abcd",
				FromAddress:  "no-reply@example.com",
				ToAddress:    "me@example.com",
			},
		},
		{
			description:      "no cert",
			shouldRaiseError: true,
			userConfig: UserConfig{
				RelayAddress: "smtp://localhost:587",
				Key:          []byte(testKey),
				Username:     "user1",
				Password:     "1234abcd",
				FromAddress:  "no-reply@example.com",
				ToAddress:    "me@example.com",
			},
		},
		{
			description:      "no username",
			shouldRaiseError: true,
			userConfig: UserConfig{
				RelayAddress: "smtp://localhost:587",
				Key:          []byte(testKey),
				Cert:         []byte(testCert),
				Password:     "1234abcd",
				FromAddress:  "no-reply@example.com",
				ToAddress:    "me@example.com",
			},
		},
		{
			description:      "no password",
			shouldRaiseError: true,
			userConfig: UserConfig{
				RelayAddress: "smtp://localhost:587",
				Key:          []byte(testKey),
				Cert:         []byte(testCert),
				Username:     "user1",
				FromAddress:  "no-reply@example.com",
				ToAddress:    "me@example.com",
			},
		},
		{
			description:      "no from address",
			shouldRaiseError: true,
			userConfig: UserConfig{
				RelayAddress: "smtp://localhost:587",
				Key:          []byte(testKey),
				Cert:         []byte(testCert),
				Username:     "user1",
				Password:     "1234abcd",
				ToAddress:    "me@example.com",
			},
		},
		{
			description:      "no to address",
			shouldRaiseError: true,
			userConfig: UserConfig{
				RelayAddress: "smtp://localhost:587",
				Key:          []byte(testKey),
				Cert:         []byte(testCert),
				Username:     "user1",
				Password:     "1234abcd",
				FromAddress:  "no-reply@example.com",
			},
		},
		{
			description:      "bad relay address",
			shouldRaiseError: true,
			userConfig: UserConfig{
				// newline character
				RelayAddress: string(rune(0x0a)),
				Key:          []byte(testKey),
				Cert:         []byte(testCert),
				Username:     "user1",
				Password:     "1234abcd",
				FromAddress:  "no-reply@example.com",
				ToAddress:    "me@example.com",
			},
		},
	}

	for _, tc := range testCases {
		_, err := NewSMTPClient(tc.userConfig)
		if (err != nil) != tc.shouldRaiseError {
			t.Errorf("%v: expected error status %v but got %v with error %v",
				tc.description,
				tc.shouldRaiseError,
				err != nil,
				err,
			)
		}
	}

}
