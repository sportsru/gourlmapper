package main

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strings"
	"time"
)

const (
	PREFIX = 0
	CODE   = 1
	FROM   = 2
	TO     = 3
)
const FILE_FIELDS_COUNT = TO + 1

type storage map[string]RedirData

// prefix -> storage
type storagesMap map[string]storage

var localStorage storagesMap

/*
{
	"ua": nil, // storageMap{},
	"ru": nil, // storageMap{},
}
*/

// global storage
//var urlMap = storageMap{}

func readUrlMapFileWorker(file string) {
	startTime := time.Now()
	defer func() {
		delta := time.Now().Sub(startTime)
		ms := int64(delta / time.Millisecond)
		log.Println("read file time ", ms, "ms (millisecons)")
	}()
	log.Println("start read file:", file)
	result, err := readUrlMapFile(file)
	if err != nil {
		log.Println("ERROR READ LOCAL STORAGE", file, err)
	} else {
		localStorage = result
	}

}

// Warning: x factor ~ 5 (file 100Mb => file 500Mb)
// for 0.5M x 2 urls ~> file = 75Mb, binary ~ 500Mb
func readUrlMapFile(file string) (storagesMap, error) {
	tmpStorage := storagesMap{}
	for _, host := range hostsMap {
		_, found := tmpStorage[host]
		if !found {
			tmpStorage[host] = storage{}
		}
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	i := 1
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		commentIdx := strings.Index(line, "#")
		if commentIdx != -1 {
			line = string(line[0:commentIdx])
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		fields := strings.Split(line, " ")
		if len(fields) != FILE_FIELDS_COUNT {
			return nil, errors.New("wrong line format:")
		}
		//log.Printf("fields %+v: %v\n", fields, len(fields))

		storage, found := tmpStorage[fields[PREFIX]]
		if !found {
			return nil, errors.New(string(i) + ": wrong line format not found prefix:" + fields[PREFIX])
		}

		storage[fields[FROM]] = RedirData{
			code: fields[CODE],
			url:  fields[TO],
		}
		//fmt.Println(i, line)
		//fmt.Println(i, scanner.Text())
		i++
	}
	return tmpStorage, nil
}
