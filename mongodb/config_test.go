package mongodb

import (
	"reflect"
	"testing"
)

// TestBuildCredential covers the auth credential assembled from the provider
// config across the supported mechanisms. It exercises the construction logic
// only (no live connection), which is the coverage available for mechanisms
// that CI cannot exercise against community mongo (X.509/AWS/OIDC).
func TestBuildCredential(t *testing.T) {
	cases := []struct {
		name        string
		cfg         ClientConfig
		wantOK      bool
		wantMech    string
		wantUser    string
		wantPass    string
		wantPassSet bool
		wantSource  string
		wantProps   map[string]string
	}{
		{
			name:   "no mechanism and no credentials leaves connection unauthenticated",
			cfg:    ClientConfig{DB: "admin"},
			wantOK: false,
		},
		{
			name:        "username and password with no mechanism (SCRAM negotiated)",
			cfg:         ClientConfig{DB: "admin", Username: "root", Password: "secret"},
			wantOK:      true,
			wantUser:    "root",
			wantPass:    "secret",
			wantPassSet: true,
			wantSource:  "admin",
		},
		{
			name:        "explicit SCRAM-SHA-256 mechanism",
			cfg:         ClientConfig{DB: "admin", Username: "root", Password: "secret", AuthMechanism: "SCRAM-SHA-256"},
			wantOK:      true,
			wantMech:    "SCRAM-SHA-256",
			wantUser:    "root",
			wantPass:    "secret",
			wantPassSet: true,
			wantSource:  "admin",
		},
		{
			name:       "MONGODB-X509 authenticates with no password",
			cfg:        ClientConfig{DB: "$external", AuthMechanism: "MONGODB-X509"},
			wantOK:     true,
			wantMech:   "MONGODB-X509",
			wantSource: "$external",
		},
		{
			name: "MONGODB-OIDC with mechanism properties and no password",
			cfg: ClientConfig{
				DB:            "$external",
				AuthMechanism: "MONGODB-OIDC",
				AuthMechanismProperties: map[string]string{
					"ENVIRONMENT":    "gcp",
					"TOKEN_RESOURCE": "https://mongodb.example",
				},
			},
			wantOK:     true,
			wantMech:   "MONGODB-OIDC",
			wantSource: "$external",
			wantProps: map[string]string{
				"ENVIRONMENT":    "gcp",
				"TOKEN_RESOURCE": "https://mongodb.example",
			},
		},
		{
			name:       "MONGODB-AWS with only a username (access key id)",
			cfg:        ClientConfig{DB: "$external", AuthMechanism: "MONGODB-AWS", Username: "AKIAEXAMPLE"},
			wantOK:     true,
			wantMech:   "MONGODB-AWS",
			wantUser:   "AKIAEXAMPLE",
			wantSource: "$external",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.cfg
			cred, ok := buildCredential(&cfg)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if cred.AuthMechanism != tc.wantMech {
				t.Errorf("AuthMechanism = %q, want %q", cred.AuthMechanism, tc.wantMech)
			}
			if cred.Username != tc.wantUser {
				t.Errorf("Username = %q, want %q", cred.Username, tc.wantUser)
			}
			if cred.Password != tc.wantPass {
				t.Errorf("Password = %q, want %q", cred.Password, tc.wantPass)
			}
			if cred.PasswordSet != tc.wantPassSet {
				t.Errorf("PasswordSet = %v, want %v", cred.PasswordSet, tc.wantPassSet)
			}
			if cred.AuthSource != tc.wantSource {
				t.Errorf("AuthSource = %q, want %q", cred.AuthSource, tc.wantSource)
			}
			if tc.wantProps != nil && !reflect.DeepEqual(cred.AuthMechanismProperties, tc.wantProps) {
				t.Errorf("AuthMechanismProperties = %v, want %v", cred.AuthMechanismProperties, tc.wantProps)
			}
		})
	}
}
