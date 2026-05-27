/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package network_drive

import (
	"context"
	"fmt"
	"net"

	"github.com/hirochachacha/go-smb2"
)

type connectionTester struct{}

func (t *connectionTester) TestConnection(ctx context.Context, config map[string]interface{}) error {
	endpoint, _ := config["endpoint"].(string)
	share, _ := config["share"].(string)
	username, _ := config["username"].(string)
	password, _ := config["password"].(string)
	domain, _ := config["domain"].(string)

	if endpoint == "" || share == "" || username == "" {
		return fmt.Errorf("endpoint, share, and username are required")
	}

	conn, err := net.DialTimeout("tcp", endpoint, ConnectionTimeout)
	if err != nil {
		return fmt.Errorf("failed to dial SMB server %s: %w", endpoint, err)
	}
	defer conn.Close()

	dialer := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     username,
			Password: password,
			Domain:   domain,
		},
	}

	dialCtx, cancel := context.WithTimeout(ctx, ConnectionTimeout)
	defer cancel()

	session, err := dialer.DialContext(dialCtx, conn)
	if err != nil {
		return fmt.Errorf("failed to authenticate with SMB server %s: %w", endpoint, err)
	}
	defer session.Logoff()

	s, err := session.Mount(share)
	if err != nil {
		return fmt.Errorf("failed to mount share '%s': %w", share, err)
	}
	defer s.Umount()

	return nil
}
