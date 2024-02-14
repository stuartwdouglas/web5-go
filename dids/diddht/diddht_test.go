package diddht

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"io"

	"github.com/alecthomas/assert/v2"
	"github.com/tbd54566975/web5-go/crypto"
	"github.com/tbd54566975/web5-go/crypto/dsa"
	"github.com/tbd54566975/web5-go/dids/didcore"
	"github.com/tbd54566975/web5-go/dids/diddht/internal/bep44"
	"github.com/tbd54566975/web5-go/dids/diddht/internal/pkarr"
	"golang.org/x/net/dns/dnsmessage"
)

type DHTTXTResourceOpt func() dnsmessage.Resource

func WithDNSRecord(name, body string) DHTTXTResourceOpt {
	return func() dnsmessage.Resource {
		return dnsmessage.Resource{
			Header: dnsmessage.ResourceHeader{
				Name: dnsmessage.MustNewName(name),
				Type: dnsmessage.TypeTXT,
				TTL:  7200,
			},
			Body: &dnsmessage.TXTResource{
				TXT: []string{
					body,
				},
			},
		}
	}
}
func makeDNSMessage(answersOpt ...DHTTXTResourceOpt) dnsmessage.Message {

	answers := []dnsmessage.Resource{}
	for _, a := range answersOpt {
		answers = append(answers, a())
	}

	msg := dnsmessage.Message{
		Header:  dnsmessage.Header{Response: true, Authoritative: true},
		Answers: answers,
	}

	return msg
}

func TestDHTResolve(t *testing.T) {
	// vector taken from https://github.com/TBD54566975/web5-js/blob/dids-new-crypto/packages/crypto/tests/fixtures/test-vectors/secp256k1/bytes-to-public-key.json
	publicKeyHexSecp256k1 := "0479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8"
	pubKeyBytesSecp256k1, err := hex.DecodeString(publicKeyHexSecp256k1)
	assert.NoError(t, err)
	base64EncodedSecp256k := base64.RawURLEncoding.EncodeToString(pubKeyBytesSecp256k1)

	keyManager := crypto.NewLocalKeyManager()
	privateKeyID, err := keyManager.GeneratePrivateKey(dsa.AlgorithmIDED25519)
	assert.NoError(t, err)

	tests := map[string]struct {
		didURI               string
		msg                  dnsmessage.Message
		expectedErrorMessage string
		assertResult         func(t *testing.T, d *didcore.Document)
		signer               bep44.Signer
	}{
		"did with valid key and no service": {
			didURI: "did:dht:cwxob5rbhhu3z9x3gfqy6cthqgm6ngrh4k8s615n7pw11czoq4fy",
			msg: makeDNSMessage(
				WithDNSRecord("_did.", "vm=k0;auth=k0;asm=k0;inv=k0;del=k0"),
				WithDNSRecord("_k0._did.", "id=0;t=0;k=YCcHYL2sYNPDlKaALcEmll2HHyT968M4UWbr-9CFGWE"),
			),

			assertResult: func(t *testing.T, d *didcore.Document) {
				t.Helper()
				assert.False(t, d == nil, "Expected non nil document")
				assert.NotZero(t, d.ID, "Expected DID Document ID to be initialized")
				assert.NotZero(t, d.VerificationMethod, "Expected at least 1 verification method")
			},
			signer: func(payload []byte) ([]byte, error) {
				return keyManager.Sign(privateKeyID, payload)
			},
		},
		"did with multiple valid keys and no service - out of order verification methods": {
			didURI: "did:dht:cwxob5rbhhu3z9x3gfqy6cthqgm6ngrh4k8s615n7pw11czoq4fy",
			msg: makeDNSMessage(
				WithDNSRecord("_did.", "vm=k0,k1,k2;auth=k0;asm=k1;inv=k2;del=k0"),
				WithDNSRecord("_k0._did.", "id=0;t=0;k=YCcHYL2sYNPDlKaALcEmll2HHyT968M4UWbr-9CFGWE"),
				WithDNSRecord("_k2._did.", fmt.Sprintf("id=2;t=1;k=%s", base64EncodedSecp256k)), //nolint:perfsprint
				WithDNSRecord("_k1._did.", fmt.Sprintf("id=1;t=1;k=%s", base64EncodedSecp256k)), //nolint:perfsprint
			),

			assertResult: func(t *testing.T, d *didcore.Document) {
				t.Helper()
				assert.False(t, d == nil, "Expected non nil document")
				assert.NotZero(t, d.ID, "Expected DID Document ID to be initialized")
				assert.Equal[int](t, 3, len(d.VerificationMethod), "Expected 3 verification methods")
			},
			signer: func(payload []byte) ([]byte, error) {
				return keyManager.Sign(privateKeyID, payload)
			},
		},
		"did with key controller and services": {
			didURI: "did:dht:cwxob5rbhhu3z9x3gfqy6cthqgm6ngrh4k8s615n7pw11czoq4fy",
			msg: makeDNSMessage(
				WithDNSRecord("_did.", "vm=k0;auth=k0;asm=k0;inv=k0;del=k0;srv=s0,s1"),
				WithDNSRecord("_k0._did.", "id=0;t=0;k=YCcHYL2sYNPDlKaALcEmll2HHyT968M4UWbr-9CFGWE"),
				WithDNSRecord("_s0._did.", "id=domain;t=LinkedDomains;se=http://foo.com"),
				WithDNSRecord("_s1._did.", "id=dwn;t=DecentralizedWebNode;se=https://dwn.tbddev.org/dwn5"),
			),

			assertResult: func(t *testing.T, d *didcore.Document) {
				t.Helper()
				assert.False(t, d == nil, "Expected non nil document")
				assert.NotZero(t, d.ID, "Expected DID Document ID to be initialized")
				assert.NotZero(t, d.VerificationMethod, "Expected at least 1 verification method")
				assert.Equal[int](t, 2, len(d.Service), "Expected 2 services")
			},
			signer: func(payload []byte) ([]byte, error) {
				return keyManager.Sign(privateKeyID, payload)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			buf, err := test.msg.Pack()
			assert.NoError(t, err)
			// test setup
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pkey, err := keyManager.GetPublicKey(privateKeyID)
				assert.NoError(t, err)
				publicKeyBytes, err := dsa.PublicKeyToBytes(pkey)
				assert.NoError(t, err)
				// create signed bep44 message
				msg, err := bep44.NewSignedBEP44Message(buf, 0, publicKeyBytes, test.signer)
				assert.NoError(t, err)

				body, _ := msg.Encode()

				// send signed bep44 message
				_, err = w.Write(body)
				assert.NoError(t, err)
			}))
			defer ts.Close()
			r := NewResolver(ts.URL, http.DefaultClient)
			result, err := r.Resolve(test.didURI)

			assert.EqualError(t, err, test.expectedErrorMessage)

			test.assertResult(t, &result.Document)
		})

	}
}

func Test_parseDNSDID(t *testing.T) {
	tests := map[string]struct {
		msg           dnsmessage.Message
		expectedError string
		assertResult  func(t *testing.T, d *Decoder)
	}{
		"basic did with key": {
			msg: makeDNSMessage(
				WithDNSRecord("_did.", "vm=k0;auth=k0;asm=k0;inv=k0;del=k0"),
				WithDNSRecord("_k0._did.", "id=0;t=0;k=YCcHYL2sYNPDlKaALcEmll2HHyT968M4UWbr-9CFGWE"),
			),
			assertResult: func(t *testing.T, d *Decoder) {
				t.Helper()
				assert.False(t, d == nil)
				expectedRecords := map[string]string{
					"_k0._did.": "id=0;t=0;k=YCcHYL2sYNPDlKaALcEmll2HHyT968M4UWbr-9CFGWE",
				}
				assert.Equal(t, "vm=k0;auth=k0;asm=k0;inv=k0;del=k0", d.rootRecord)
				assert.True(t, reflect.DeepEqual(expectedRecords, d.records))
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			buf, err := test.msg.Pack()
			assert.NoError(t, err)

			dhtDidRecord, err := parseDNSDID(buf)
			assert.EqualError(t, err, test.expectedError)

			assert.Equal(t, "vm=k0;auth=k0;asm=k0;inv=k0;del=k0", dhtDidRecord.rootRecord)

		})
	}
}

func Test_Create(t *testing.T) {
	tests := map[string]struct {
		didURI         string
		expectedResult string
		didDocData     string
	}{
		"": {
			didURI:         "did:dht:1wiaaaoagzceggsnwfzmx5cweog5msg4u536mby8sqy3mkp3wyko",
			expectedResult: "1wiaaaoagzceggsnwfzmx5cweog5msg4u536mby8sqy3mkp3wyko",
			didDocData: `{
				"id": "did:dht:1wiaaaoagzceggsnwfzmx5cweog5msg4u536mby8sqy3mkp3wyko",
				"verificationMethod": [
				  {
					"id": "did:dht:1wiaaaoagzceggsnwfzmx5cweog5msg4u536mby8sqy3mkp3wyko#0",
					"type": "JsonWebKey2020",
					"controller": "did:dht:1wiaaaoagzceggsnwfzmx5cweog5msg4u536mby8sqy3mkp3wyko",
					"publicKeyJwk": {
					  "kty": "OKP",
					  "crv": "Ed25519",
					  "x": "lSuMYhg12IMawqFut-2URA212Nqe8-WEB7OBlam5oBU",
					  "kid": "2Jr7faCpoEgHvy5HXH32z-MH_0CRToO9NllZtemVvNo"
					}
				  }
				],
				"service": [
				  {
					"id": "did:dht:1wiaaaoagzceggsnwfzmx5cweog5msg4u536mby8sqy3mkp3wyko#dwn",
					"type": "DecentralizedWebNode",
					"serviceEndpoint": "https://example.com/dwn"
				  }
				],
				"authentication": [
				  "did:dht:1wiaaaoagzceggsnwfzmx5cweog5msg4u536mby8sqy3mkp3wyko#0"
				],
				"assertionMethod": [
				  "did:dht:1wiaaaoagzceggsnwfzmx5cweog5msg4u536mby8sqy3mkp3wyko#0"
				],
				"capabilityDelegation": [
				  "did:dht:1wiaaaoagzceggsnwfzmx5cweog5msg4u536mby8sqy3mkp3wyko#0"
				],
				"capabilityInvocation": [
				  "did:dht:1wiaaaoagzceggsnwfzmx5cweog5msg4u536mby8sqy3mkp3wyko#0"
				]
			  }`,
		},
	}

	// setting up a fake relay that stores did documents on publish, and responds with the bencoded did document on resolve
	mockedRes := map[string][]byte{}
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did := fmt.Sprintf("did:dht:%s", r.URL.Path[1:])
		defer r.Body.Close()

		// create branch
		if r.Method != http.MethodGet {
			packagedDid, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			mockedRes[did] = packagedDid
			w.WriteHeader(http.StatusOK)
			return
		}

		// resolve branch
		expectedBuf, ok := mockedRes[did]
		if !ok {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Write(expectedBuf)
	}))

	defer relay.Close()
	relayClient := pkarr.NewPkarrRelay(relay.URL, http.DefaultClient)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var didDoc didcore.Document
			assert.NoError(t, json.Unmarshal([]byte(test.didDocData), &didDoc))
			keyMgr := crypto.NewLocalKeyManager()
			createdDid, err := Create(
				WithVerificationMethod(didDoc.VerificationMethod[0], []didcore.Purpose{didcore.PurposeAssertion, didcore.PurposeAuthentication, didcore.PurposeCapabilityDelegation, didcore.PurposeCapabilityInvocation}),
				WithServices(didDoc.Service...),
				WithRelay(relayClient),
				WithKeyManager(keyMgr),
			)
			assert.NoError(t, err)
			resolver := NewResolver(relay.URL, http.DefaultClient)
			result, err := resolver.Resolve(createdDid.URI)
			assert.NoError(t, err)
			assert.Equal(t, createdDid.Document.Authentication, result.Document.Authentication)
			assert.Equal(t, createdDid.Document.AssertionMethod, result.Document.AssertionMethod)
			assert.Equal(t, createdDid.Document.CapabilityDelegation, result.Document.CapabilityDelegation)
			assert.Equal(t, createdDid.Document.CapabilityInvocation, result.Document.CapabilityInvocation)
			assert.Equal(t, createdDid.Document.Service, result.Document.Service)
			assert.Equal(t, createdDid.Document.VerificationMethod, result.Document.VerificationMethod)
		})
	}
}
