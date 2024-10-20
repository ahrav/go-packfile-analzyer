package scanner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
)

// PackScanner is the main struct for scanning packfiles.
type PackScanner struct {
	RepoURL string
	Wants   []plumbing.Hash
	Haves   []plumbing.Hash
}

// NewPackScanner creates a new instance of PackScanner.
// It validates the repository URL and converts string hashes to plumbing.Hash.
func NewPackScanner(repoURL string, wants []string, haves []string) (*PackScanner, error) {
	// Validate and parse the repository URL.
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL %s: %w", repoURL, err)
	}

	// Ensure the URL has a supported scheme.
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" && parsedURL.Scheme != "ssh" {
		return nil, fmt.Errorf("unsupported URL scheme %s", parsedURL.Scheme)
	}

	// Convert wants and haves to plumbing.Hash.
	var wantHashes, haveHashes []plumbing.Hash
	for _, w := range wants {
		wantHashes = append(wantHashes, plumbing.NewHash(w))
	}

	for _, h := range haves {
		haveHashes = append(haveHashes, plumbing.NewHash(h))
	}

	return &PackScanner{
		RepoURL: repoURL,
		Wants:   wantHashes,
		Haves:   haveHashes,
	}, nil
}

// PackfileReader implements the io.Reader interface to stream raw Git object bytes.
type PackfileReader struct{ pipeReader *io.PipeReader }

// Read implements the io.Reader interface for PackfileReader.
func (pfr *PackfileReader) Read(p []byte) (n int, err error) {
	return pfr.pipeReader.Read(p)
}

// ScanPackfile initializes and returns a PackfileReader.
// It starts a goroutine to fetch and parse the packfile, streaming the raw bytes.
func (ps *PackScanner) ScanPackfile(ctx context.Context) (*PackfileReader, error) {
	// Create a pipe for streaming data.
	pipeR, pipeW := io.Pipe()

	pfr := &PackfileReader{pipeReader: pipeR}

	// Start the processing in a separate goroutine.
	go ps.processPackfile(ctx, pipeW)

	return pfr, nil
}

// processPackfile handles fetching and parsing the packfile,
// writing raw Git object bytes directly to the pipe's writer.
func (ps *PackScanner) processPackfile(ctx context.Context, pipeW *io.PipeWriter) {
	defer pipeW.Close()

	// Fetch the packfile data as an io.Reader.
	packReader, err := ps.fetchPackfile(ctx)
	if err != nil {
		_ = pipeW.CloseWithError(fmt.Errorf("failed to fetch packfile: %w", err))
		return
	}
	defer packReader.Close()

	// Parse the packfile and write raw bytes to the pipe.
	err = ps.parsePackfile(ctx, packReader, pipeW)
	if err != nil {
		_ = pipeW.CloseWithError(fmt.Errorf("failed to parse packfile: %w", err))
		return
	}
}

// fetchPackfile retrieves the packfile data from the repository and returns an io.Reader.
// It respects the provided context for cancellation and timeouts.
func (ps *PackScanner) fetchPackfile(ctx context.Context) (io.ReadCloser, error) {
	endpoint, err := transport.NewEndpoint(ps.RepoURL)
	if err != nil {
		return nil, fmt.Errorf("error creating endpoint: %w", err)
	}

	c, err := client.NewClient(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	session, err := c.NewUploadPackSession(endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating session: %w", err)
	}
	defer session.Close()

	req := packp.NewUploadPackRequest()
	req.Wants = append(req.Wants, ps.Wants...)
	if len(ps.Haves) > 0 {
		req.Haves = append(req.Haves, ps.Haves...)
	}

	// Enable capabilities.
	if err := req.Capabilities.Add(capability.ThinPack); err != nil {
		return nil, fmt.Errorf("error adding ThinPack capability: %w", err)
	}
	if err := req.Capabilities.Add(capability.OFSDelta); err != nil {
		return nil, fmt.Errorf("error adding OFSDelta capability: %w", err)
	}

	resp, err := session.UploadPack(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error sending upload-pack request: %w", err)
	}

	return resp, nil
}

// parsePackfile parses the packfile data from an io.Reader and writes raw GitObject bytes to the writer.
func (ps *PackScanner) parsePackfile(ctx context.Context, r io.Reader, writer io.Writer) error {
	packfileReader := packfile.NewScanner(r)

	_, objCnt, err := packfileReader.Header()
	if err != nil {
		return fmt.Errorf("error reading packfile header: %w", err)
	}

	for i := uint32(0); i < objCnt; i++ {
		// Check for cancellation.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

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

			// Write the raw bytes of the object directly to the writer.
			_, err = writer.Write(objBuf.Bytes())
			if err != nil {
				return fmt.Errorf("error writing to writer: %w", err)
			}
		}
	}

	return nil
}
