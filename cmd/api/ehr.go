package main

import (
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/gofrs/uuid"
)

func (s *storage) saveFileWithChecks(owner, account, key, signature string, file multipart.File, header *multipart.FileHeader, reuploadFid string) (fid string, ts string, code int, err error) {
	if code, err = s.checkEOSAccountConnections(account, owner, key); err != nil {
		return "", "", code, err
	}

	return s.saveFile(owner, account, key, signature, file, header, reuploadFid)
}
func (s *storage) saveFile(owner, account, key, signature string, file multipart.File, header *multipart.FileHeader, reuploadFid string) (fid string, ts string, code int, err error) {
	if reuploadFid != "" {
		if _, err = os.Stat(fmt.Sprintf("%s/%s/%s/%s", s.config.StoragePath, "ehr", owner, fid)); os.IsNotExist(err) {
			return "", "", 404, fmt.Errorf("File %s not found, cannot overwrite", fid)
		}
		fid = reuploadFid
	} else {
		fid, code, err = s.getFilename(header)
		if err != nil {
			return "", "", code, err
		}
	}

	data, code, err := s.getUploadedFile(key, signature, file, header)
	if err != nil {
		return "", "", code, err
	}

	os.MkdirAll(fmt.Sprintf("%s/%s/%s", s.config.StoragePath, "ehr", owner), os.ModePerm)

	// create file
	f, err := os.Create(fmt.Sprintf("%s/%s/%s/%s", s.config.StoragePath, "ehr", owner, fid))
	if err != nil {
		s.log.Printf("Failed to create new file. Error: %+v", err)
		return "", "", 500, fmt.Errorf("Failed to create new file")
	}
	defer f.Close()

	// add data to file
	_, err = f.WriteString(string(data))
	if err != nil {
		s.log.Printf("Failed to save data to file. Error: %+v", err)
		return "", "", 500, fmt.Errorf("Failed to save data to file")
	}
	// Get the timestamp
	ts, code, err = s.getFileTimestamp(fid)
	s.log.Debugf("File %s uploaded", fid)
	return
}

func (s *storage) getFilename(header *multipart.FileHeader) (fid string, code int, err error) {
	uuid, err := uuid.NewV1()
	if err != nil {
		s.log.Printf("Failed to create filename")
		return "", 500, fmt.Errorf("Error creating filename")
	}
	fid = uuid.String()
	return fid, 200, nil
}

func (s *storage) getUploadedFile(keystr, signature string, file multipart.File, header *multipart.FileHeader) ([]byte, int, error) {
	data := make([]byte, header.Size)
	if _, err := file.Read(data); err != nil {
		s.log.Debugf("Problem reading uploaded file. Error: %+v", err)
		return []byte{}, 500, fmt.Errorf("Internal server problem reading file")
	}
	if code, err := s.checkSignature(keystr, signature, data); err != nil {
		return []byte{}, code, err
	}
	return data, 200, nil
}

func (s *storage) getFileTimestamp(name string) (string, int, error) {
	id, err := uuid.FromString(name)
	if err != nil {
		s.log.Printf("Failed to create uuid from string. Err; %+v", err)
		return "", 500, fmt.Errorf("Internal server error")
	}

	ts, err := uuid.TimestampFromV1(id)
	if err != nil {
		s.log.Debugf("Failed to get timestamp from uuid. Err; %+v", err)
		return "", 500, fmt.Errorf("Internal server error")
	}

	t, err := ts.Time()
	if err != nil {
		s.log.Debugf("Failed to get time from uuid timestamp. Err; %+v", err)
		return "", 500, fmt.Errorf("Internal server error")
	}
	return t.Format("2006-01-02T15:04:05.999Z"), 200, nil
}

type lsResponse struct {
	Files []lsFile `json:"files,omitempty"`
}

type lsFile struct {
	FileID    string `json:"fileID,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

func (s *storage) listFiles(account string) (*lsResponse, int, error) {
	out := &lsResponse{}

	files, err := filepath.Glob(fmt.Sprintf("%s/%s/%s/*", s.config.StoragePath, "ehr", account))
	if err != nil {
		s.log.Debugf("Error getting list of files; %+v", err)
		return out, 500, fmt.Errorf("Internal server error. Failed getting list of files")
	} else {
		for _, f := range files {
			fn := filepath.Base(f)
			ts, code, err := s.getFileTimestamp(fn)
			if err != nil {
				return out, code, err
			}
			out.Files = append(out.Files, lsFile{fn, ts})
		}
		if len(out.Files) == 0 {
			return out, 404, fmt.Errorf("404 no files found for account %s", account)
		}
	}
	return out, 200, nil
}

func (s *storage) readFileData(account, fid string) ([]byte, int, error) {
	// Check that dir exits
	if _, err := os.Stat(fmt.Sprintf("%s/%s/%s", s.config.StoragePath, "ehr", account)); !os.IsNotExist(err) {
		// Check that file exists
		_, err := os.Stat(fmt.Sprintf("%s/%s/%s/%s", s.config.StoragePath, "ehr", account, fid))
		if err == nil {
			f, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/%s/%s", s.config.StoragePath, "ehr", account, fid))
			if err != nil {
				s.log.Debugf("Error reading file %s/%s. Err: %+v", account, fid, err)
				return nil, 500, fmt.Errorf("Internal server error")
			}
			return f, 200, nil
		} else {
			s.log.Debugf("Error checking/reading/getting file %s/%s. Err; %+v", account, fid, err)
			return nil, 404, fmt.Errorf("404 file not found")
		}
	} else {
		s.log.Debugf("Folder %s not found. Err: %+v", account, err)
		return nil, 404, fmt.Errorf("404 account's folder does not exits")
	}
}
