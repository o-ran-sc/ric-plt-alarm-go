/*
==================================================================================
  Copyright (c) 2019 AT&T Intellectual Property.
  Copyright (c) 2019 Nokia

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
==================================================================================
*/

package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	app "gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
)

type Utils struct {
	baseDir string
	status  string
}

func NewUtils() *Utils {
	b := app.Config.GetString("controls.symptomdata.baseDir")
	if b == "" {
		b = "/tmp/symptomdata/"
	}

	return &Utils{
		baseDir: b,
	}
}

func (u *Utils) FileExists(f string) bool {
	_, err := os.Stat(f)
	return err == nil || os.IsExist(err)
}

func (u *Utils) CreateDir(path string) error {
	if u.FileExists(path) {
		os.RemoveAll(path)
	}
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}
	os.Chmod(path, os.ModePerm)
	return nil
}

func (u *Utils) DeleteFile(fileName string) {
	os.Remove(fileName)
}

func (u *Utils) AddFileToZip(zipWriter *zip.Writer, filePath string, filename string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	if strings.HasPrefix(filename, filePath) {
		filename = strings.TrimPrefix(filename, filePath)
	}
	header.Name = filename
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	if info.Size() > 0 {
		_, err = io.Copy(writer, fileToZip)
	}
	return err
}

func (u *Utils) ZipFiles(newZipFile *os.File, filePath string, files []string) error {
	defer newZipFile.Close()
	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()
	for _, file := range files {
		if err := u.AddFileToZip(zipWriter, filePath, file); err != nil {
			app.Logger.Error("AddFileToZip() failed: %+v", err.Error())
			return err
		}
	}

	return nil
}

func (u *Utils) FetchFiles(filePath string, fileList []string) []string {
	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		app.Logger.Error("ioutil.ReadDir failed: %+v", err)
		return nil
	}
	for _, file := range files {
		if !file.IsDir() {
			fileList = append(fileList, filepath.Join(filePath, file.Name()))
		} else {
			subPath := filepath.Join(filePath, file.Name())
			subFiles, _ := ioutil.ReadDir(subPath)
			for _, subFile := range subFiles {
				if !subFile.IsDir() {
					fileList = append(fileList, filepath.Join(subPath, subFile.Name()))
				} else {
					fileList = u.FetchFiles(filepath.Join(subPath, subFile.Name()), fileList)
				}
			}
		}
	}
	return fileList
}

func (u *Utils) WriteToFile(fileName string, data string) error {
	f, err := os.Create(fileName)
	defer f.Close()

	if err != nil {
		app.Logger.Error("Unable to create file %s': %+v", fileName, err)
	} else {
		_, err := io.WriteString(f, data)
		if err != nil {
			app.Logger.Error("Unable to write to file '%s': %+v", fileName, err)
			u.DeleteFile(fileName)
		}
	}
	return err
}

func (u *Utils) SendSymptomDataJson(w http.ResponseWriter, req *http.Request, data interface{}, n string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename="+n)
	w.WriteHeader(http.StatusOK)
	if data != nil {
		response, _ := json.Marshal(data)
		w.Write(response)
	}
}

func (u *Utils) SendSymptomDataFile(w http.ResponseWriter, req *http.Request, baseDir, zipFile string) {
	// Compress and reply with attachment
	tmpFile, err := ioutil.TempFile("", "symptom")
	if err != nil {
		u.SendSymptomDataError(w, req, "Failed to create a tmp file: "+err.Error())
		return
	}
	defer os.Remove(tmpFile.Name())

	var fileList []string
	fileList = u.FetchFiles(baseDir, fileList)
	err = u.ZipFiles(tmpFile, baseDir, fileList)
	if err != nil {
		u.SendSymptomDataError(w, req, "Failed to zip the files: "+err.Error())
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+zipFile)
	http.ServeFile(w, req, tmpFile.Name())
}

func (u *Utils) SendSymptomDataError(w http.ResponseWriter, req *http.Request, message string) {
	w.Header().Set("Content-Disposition", "attachment; filename=error_status.txt")
	http.Error(w, message, http.StatusInternalServerError)
}
