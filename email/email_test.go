package email

import (
	"divnews/testutil"
	"testing"
)

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
				Key:          []byte(testutil.TestKey),
				Cert:         []byte(testutil.TestCert),
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
				Key:          []byte(testutil.TestKey),
				Cert:         []byte(testutil.TestCert),
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
				Key:         []byte(testutil.TestKey),
				Cert:        []byte(testutil.TestCert),
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
				Cert:         []byte(testutil.TestCert),
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
				Key:          []byte(testutil.TestKey),
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
				Key:          []byte(testutil.TestKey),
				Cert:         []byte(testutil.TestCert),
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
				Key:          []byte(testutil.TestKey),
				Cert:         []byte(testutil.TestCert),
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
				Key:          []byte(testutil.TestKey),
				Cert:         []byte(testutil.TestCert),
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
				Key:          []byte(testutil.TestKey),
				Cert:         []byte(testutil.TestCert),
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
				Key:          []byte(testutil.TestKey),
				Cert:         []byte(testutil.TestCert),
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
