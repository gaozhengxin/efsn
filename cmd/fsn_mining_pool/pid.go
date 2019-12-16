package main

import (
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"
)

type PIDFile struct {
    path string
}

// just suit for linux
func processExists(pid string) bool {
    if _, err := os.Stat(filepath.Join("/proc", pid)); err == nil {
        return true
    }
    return false
}

func checkPIDFILEAlreadyExists(path string) error {
    if pidByte, err := ioutil.ReadFile(path); err == nil {
        pid := strings.TrimSpace(string(pidByte))
        if processExists(pid) {
            return fmt.Errorf("ensure the process:%s is not running pid file:%s", pid, path)
        }
    }
    return nil
}

func NewPIDFile(path string) (*PIDFile, error) {
    if err := checkPIDFILEAlreadyExists(path); err != nil {
        return nil, err
    }

    if err := os.MkdirAll(filepath.Dir(path), os.FileMode(0755)); err != nil {
        return nil, err
    }
    if err := ioutil.WriteFile(path, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
        return nil, err
    }
    return &PIDFile{path: path}, nil
}

func (file PIDFile) Remove() error {
    return os.Remove(file.path)
}
