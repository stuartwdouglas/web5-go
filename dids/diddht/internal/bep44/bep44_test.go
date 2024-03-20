package bep44

import (
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/tbd54566975/web5-go/dids/diddht/internal/bencode"
)

func Test_newSignedBEP44Message(t *testing.T) {
	payload := []byte(`v=1,b=2,c=3`)

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	type args struct {
		payload        []byte
		seq            int64
		publicKeyBytes []byte
		signer         Signer
	}
	tests := map[string]struct {
		args    args
		wantErr bool
	}{
		"good - create signed message and decode payload": {
			args: args{
				payload:        payload,
				seq:            time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC).Unix() / 1000,
				publicKeyBytes: pubKey,
				signer: func(payload []byte) ([]byte, error) {
					return ed25519.Sign(privKey, payload), nil
				},
			},
		},
		"bad - signer fails": {
			args: args{
				payload:        payload,
				seq:            time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC).Unix() / 1000,
				publicKeyBytes: []byte("YCcHYL2sYNPDlKaALcEmll2HHyT968M4UWbr-9CFGWE"),
				signer: func(payload []byte) ([]byte, error) {
					return nil, errors.New("signer failed")
				},
			},
			wantErr: true,
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			got, err := NewMessage(tt.args.payload, tt.args.seq, tt.args.publicKeyBytes, tt.args.signer)
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.wantErr {
				return
			}
			assert.Equal(t, tt.args.publicKeyBytes, got.k)

			bencoded := map[string]any{
				"seq": got.Seq,
				"v":   got.V,
			}

			bencodedBytes, err := bencode.Marshal(bencoded)
			assert.NoError(t, err)
			assert.True(t, ed25519.Verify(tt.args.publicKeyBytes, bencodedBytes, got.sig))
		})
	}
}
