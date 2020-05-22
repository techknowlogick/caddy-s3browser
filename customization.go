package s3browser

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"regexp"
	"time"

	"github.com/minio/minio-go/v6"
)

const customizationFile = ".s3-customize.json"
const maxCustomizationSize = 65536
const getCustomizationTimeout = time.Second * 5

type CustomizationEntry struct {
	Icon        string
	Description string
}

type CustomizationConfig struct {
	// Customization for entries matching the given pattern
	CustomizationEntry
	// Customization for entries marked as "latest" when using semver
	Latest *CustomizationEntry
	// Information derived from the config
	Pattern  *regexp.Regexp `json:"-"`
	IsFolder bool           `json:"-"`
	IsFile   bool           `json:"-"`
}

func getCustomization(logger *log.Logger, client *S3Client, obj minio.ObjectInfo) []*CustomizationConfig {

	if obj.Size > maxCustomizationSize {
		logger.Printf("Error retrieving %s from bucket storage: size %d is bigger than permitted %d\n", obj.Key, obj.Size, maxCustomizationSize)
		return nil
	}

	reader, _, _, err := client.GetObject(obj.Key, "0-")
	if err != nil {
		logger.Printf("Error retrieving %s from bucket storage: %v\n", obj.Key, err)
		return nil
	}

	buffer := new(bytes.Buffer)
	if rd, err := buffer.ReadFrom(reader); err != nil && err != io.EOF {
		logger.Printf("Error retrieving %s from bucket storage: %v\n", obj.Key, err)
		return nil
	} else if rd != obj.Size {
		logger.Printf("Error retrieving %s from bucket storage: read %d bytes, but expected %d\n", obj.Key, rd, obj.Size)
		return nil
	}

	parsed := new(map[string]*CustomizationConfig)
	if err := json.Unmarshal(buffer.Bytes(), parsed); err != nil {
		logger.Printf("Error parsing configuration file %s: %v\n", obj.Key, err)
		return nil
	}

	if len(*parsed) == 0 {
		return nil
	}

	config := make([]*CustomizationConfig, 0, len(*parsed))
	for k, v := range *parsed {
		if k == "folder" {
			v.IsFolder = true
		} else if k == "file" {
			v.IsFile = true
		} else {
			if v.Pattern, err = regexp.Compile(k); err != nil {
				logger.Printf("Warning: regexp '%s' in %s could not be parsed: %v\n", k, obj.Key, err)
				continue
			}
		}
		config = append(config, v)
	}

	return config
}
