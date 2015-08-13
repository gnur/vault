package ssh

import (
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

type sshHostKey struct {
	Key string `json:"key"`
}

func pathKeys(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "keys/(?P<key_name>[-\\w]+)",
		Fields: map[string]*framework.FieldSchema{
			"key_name": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Name of the key",
			},
			"key": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "SSH private key with root privileges for host",
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation:   b.pathKeysRead,
			logical.WriteOperation:  b.pathKeysWrite,
			logical.DeleteOperation: b.pathKeysDelete,
		},
		HelpSynopsis:    pathKeysSyn,
		HelpDescription: pathKeysDesc,
	}
}

func (b *backend) getKey(s logical.Storage, n string) (*sshHostKey, error) {
	entry, err := s.Get("keys/" + n)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	var result sshHostKey
	if err := entry.DecodeJSON(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (b *backend) pathKeysRead(req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	key, err := b.getKey(req.Storage, d.Get("key_name").(string))
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"key": key.Key,
		},
	}, nil
}

func (b *backend) pathKeysDelete(req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	keyName := d.Get("key_name").(string)
	keyPath := fmt.Sprintf("keys/%s", keyName)
	err := req.Storage.Delete(keyPath)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (b *backend) pathKeysWrite(req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	keyName := d.Get("key_name").(string)
	keyString := d.Get("key").(string)

	signer, err := ssh.ParsePrivateKey([]byte(keyString))
	if err != nil || signer == nil {
		return logical.ErrorResponse("Invalid key"), nil
	}

	if keyString == "" {
		return logical.ErrorResponse("Missing key"), nil
	}

	keyPath := fmt.Sprintf("keys/%s", keyName)

	entry, err := logical.StorageEntryJSON(keyPath, map[string]interface{}{
		"key": keyString,
	})
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(entry); err != nil {
		return nil, err
	}
	return nil, nil
}

const pathKeysSyn = `
Register a shared key which can be used to install dynamic key
in remote machine.
`

const pathKeysDesc = `
The shared key registered will be used to install and uninstall
long lived dynamic keys in remote machine. This key should have
"root" privileges at target machine. This enables installing keys
for unprivileged usernames.

If this backend is mounted as "ssh", then the endpoint for registering
shared key is "ssh/keys/webrack", if "webrack" is the user coined 
name for the key. The name given here can be associated with any
number of roles via the endpoint "ssh/roles/".
`
