package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
)

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	repoURL := "https://github.com/trufflesecurity/trufflehog.git"
	endpoint, err := transport.NewEndpoint(repoURL)
	if err != nil {
		return fmt.Errorf("error creating endpoint: %w", err)
	}

	c, err := client.NewClient(endpoint)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	session, err := c.NewUploadPackSession(endpoint, nil)
	if err != nil {
		return fmt.Errorf("error creating session: %w", err)
	}
	defer session.Close()

	wants := []plumbing.Hash{
		plumbing.NewHash("eaceca8c2e77a8b0dae5ce976bf1901b8accd68f"),
	}
	haves := []plumbing.Hash{
		plumbing.NewHash("9edeb164f449abb91c7abd81d82ea4fe80a8ed8a"),
		// plumbing.NewHash("2a626c4daba0e8bcc96debe579e75c22149412e6"),
	}

	req := packp.NewUploadPackRequest()
	req.Wants = append(req.Wants, wants...)
	req.Haves = append(req.Haves, haves...)

	// Enable capabilities.
	if err := req.Capabilities.Add(capability.ThinPack); err != nil {
		return fmt.Errorf("error adding capability: %w", err)

	}
	if err := req.Capabilities.Add(capability.OFSDelta); err != nil {
		return fmt.Errorf("error adding capability: %w", err)
	}

	ctx := context.Background()

	resp, err := session.UploadPack(ctx, req)
	if err != nil {
		return fmt.Errorf("error sending upload-pack request: %w", err)
	}
	defer resp.Close()

	buf, err := io.ReadAll(resp)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}
	fmt.Printf("Size of packfile: %+d\n", len(buf))

	filePath := "packfile.pack"
	err = os.WriteFile(filePath, buf, 0644)
	if err != nil {
		return fmt.Errorf("error writing packfile to disk: %w", err)
	}
	fmt.Printf("Packfile saved to %s\n", filePath)

	r := bytes.NewReader(buf)
	packfileReader := packfile.NewScanner(r)

	_, objCnt, err := packfileReader.Header()
	if err != nil {
		return fmt.Errorf("error reading packfile header: %w", err)
	}

	for i := uint32(0); i < objCnt; i++ {
		hdr, err := packfileReader.NextObjectHeader()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading next object header: %w", err)
		}

		objType := hdr.Type
		if objType == plumbing.BlobObject || objType == plumbing.CommitObject {
			objBuf := new(bytes.Buffer)
			_, _, err = packfileReader.NextObject(objBuf)
			if err != nil {
				return fmt.Errorf("error reading object: %w", err)
			}

			// Process the object data
			// processObject(objType, objBuf.Bytes())
		}
	}

	return nil
}
