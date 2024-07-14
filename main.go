package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

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
	// repoURL := "https://github.com/dylanTruffle/test.git"
	repoURL := "https://github.com/trufflesecurity/trufflehog.git"
	endpoint, err := transport.NewEndpoint(repoURL)
	if err != nil {
		return fmt.Errorf("error creating endpoint: %w", err)
	}

	c, err := client.NewClient(endpoint)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Connect to the remote repository
	session, err := c.NewUploadPackSession(endpoint, nil)
	if err != nil {
		return fmt.Errorf("error creating session: %w", err)
	}
	defer session.Close()

	// Prepare the wants and haves
	wants := []plumbing.Hash{
		// plumbing.NewHash("37391c5971744ec6aac0631fe8f92a7b7b30a6e5"),
		plumbing.NewHash("eaceca8c2e77a8b0dae5ce976bf1901b8accd68f"),
	}
	haves := []plumbing.Hash{
		// plumbing.NewHash("bbd29c0b9a226c9d6a31d45285d8ea10c89e27ac"),
		plumbing.NewHash("9edeb164f449abb91c7abd81d82ea4fe80a8ed8a"),
		// plumbing.NewHash("2a626c4daba0e8bcc96debe579e75c22149412e6"),
	}

	req := packp.NewUploadPackRequest()
	req.Wants = append(req.Wants, wants...)
	req.Haves = append(req.Haves, haves...)

	// Enable capabilities
	if err := req.Capabilities.Add(capability.ThinPack); err != nil {
		return fmt.Errorf("error adding capability: %w", err)

	}
	if err := req.Capabilities.Add(capability.OFSDelta); err != nil {
		return fmt.Errorf("error adding capability: %w", err)
	}

	ctx := context.Background()

	now := time.Now()
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
	// fmt.Printf("Downloaded packfile in %v\n", time.Since(now))
	//
	// filePath := "packfile.pack"
	// err = os.WriteFile(filePath, buf, 0644)
	// if err != nil {
	// 	return fmt.Errorf("error writing packfile to disk: %w", err)
	// }
	// fmt.Printf("Packfile saved to %s\n", filePath)
	//
	// // Unpack the packfile using go-git
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

	fmt.Printf("Downloaded packfile in %v\n", time.Since(now))

	// storage := memory.NewStorage()
	// parser, err := packfile.NewParserWithStorage(packfileReader, storage)
	// if err != nil {
	// 	return fmt.Errorf("error creating packfile parser: %w", err)
	// }
	//
	// _, err = parser.Parse()
	// if err != nil {
	// 	return fmt.Errorf("error parsing packfile: %w", err)
	// }
	//
	// // Iterate over the objects in the packfile
	// iter, err := storage.IterEncodedObjects(plumbing.AnyObject)
	// if err != nil {
	// 	return fmt.Errorf("error creating object iterator: %w", err)
	// }
	//
	// err = iter.ForEach(func(obj plumbing.EncodedObject) error {
	// 	fmt.Printf("Object: %s, Size: %d, Type: %s\n", obj.Hash().String(), obj.Size(), obj.Type().String())
	//
	// 	// Print the contents of the object
	// 	objReader, err := obj.Reader()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	defer objReader.Close()
	//
	// 	switch obj.Type() {
	// 	case plumbing.CommitObject:
	// 		_, err := object.DecodeCommit(storage, obj)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		// fmt.Printf("Commit: %s\n", commit)
	// 	case plumbing.TreeObject:
	// 		// tree, err := object.DecodeTree(storage, obj)
	// 		// if err != nil {
	// 		// 	return err
	// 		// }
	// 		// fmt.Printf("Tree: %v\n", tree)
	// 	case plumbing.BlobObject:
	// 		_, err := object.DecodeBlob(obj)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		// fmt.Printf("Blob: %v\n", blob)
	// 		// fmt.Printf("Blob contents: %s\n", contents)
	// 	case plumbing.TagObject:
	// 		// tag, err := object.DecodeTag(storage, obj)
	// 		// if err != nil {
	// 		// 	return err
	// 		// }
	// 		// fmt.Printf("Tag: %v\n", tag)
	// 	default:
	// 		fmt.Printf("Unknown object type: %v\n", obj.Type())
	// 	}
	//
	// 	return nil
	// })
	// if err != nil {
	// 	return fmt.Errorf("error iterating over objects: %w", err)
	// }
	//

	return nil
}

// remote := git.NewRemote(nil, &config.RemoteConfig{
// 	Name: "origin",
// 	URLs: []string{repoURL},
// })

// List all refs
// refs, err := remote.List(&git.ListOptions{})
// if err != nil {
// 	log.Fatalf("failed to list refs: %v", err)
// }
//
// haves := make([]plumbing.Hash, 0, len(refs))
// for _, ref := range refs {
// 	haves = append(haves, ref.Hash())
// }

// req, err := http.NewRequestWithContext(ctx, "GET",
// 	"https://api.github.com/"+path.Join("repos", "trufflesecurity", "trufflehog", "compare")+"/"+"9edeb164f449abb91c7abd81d82ea4fe80a8ed8a"+"..."+"eaceca8c2e77a8b0dae5ce976bf1901b8accd68f",
// 	nil,
// )
// req.Header.Set("Accept", "application/vnd.github.diff")
// if err != nil {
// 	return err
// }
//
// httpClient := new(http.Client)
// resp, err := httpClient.Do(req)
// if err != nil {
// 	return err
// }
// defer resp.Body.Close()
//
// if resp.StatusCode == http.StatusNotFound {
// 	return errors.New("diff not found")
// }
// if resp.StatusCode == http.StatusForbidden {
// 	err := errors.New("forbidden")
// 	return err
// }
// if resp.StatusCode != http.StatusOK {
// 	err := errors.New("received non-200 status code")
// 	return err
// }
//
// _, err = io.ReadAll(resp.Body)
// if err != nil {
// 	return err
// }

// _, err := fetchDiff(ctx, "trufflesecurity", "trufflehog", "9edeb164f449abb91c7abd81d82ea4fe80a8ed8a", "eaceca8c2e77a8b0dae5ce976bf1901b8accd68f")
// if err != nil {
// 	return fmt.Errorf("error fetching diff: %w", err)
// }
