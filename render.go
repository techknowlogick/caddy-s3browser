package s3browser

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"regexp"
	"sort"
	"time"

	"github.com/Masterminds/semver"
	"github.com/minio/minio-go/v6"
)

// semVerRegex is the regular expression used to parse a partial semantic version.
// We rely on github.com/Masterminds/semver for the actual parsing, but
// we want to consider the edge cases 1.0.0 vs. 1.0 vs 1.
var semVerRegex = regexp.MustCompile(`^v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?`)

const customizationFile = ".s3-customize.json"
const maxCustomizationSize = 65536
const getCustomizationTimeout = time.Second * 5

type customizationEntry struct {
	Icon        string
	Description string
}

type customizationConfig struct {
	// Customization for entries matching the given pattern
	Default *customizationEntry
	// Customization for entries marked as "latest" when using semver
	Latest *customizationEntry
	// Information derived from the config
	Pattern         *regexp.Regexp `json:"-"`
	IsDefaultFolder bool           `json:"-"`
	IsDefaultFile   bool           `json:"-"`
}

type RenderizableItem struct {
	Name        string
	Icon        string
	Description string
}

type RenderizableDir struct {
	RenderizableItem
	version *semver.Version
	Latest  bool
}

type RenderizableFile struct {
	RenderizableItem
	File
}

type dirCollection []*RenderizableDir
type dirSemverCollection []*RenderizableDir
type fileCollection []*RenderizableFile

func getRenderCustomization(logger *log.Logger, client *S3Client, obj minio.ObjectInfo) []*customizationConfig {

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

	parsed := new(map[string]*customizationConfig)
	if err := json.Unmarshal(buffer.Bytes(), parsed); err != nil {
		logger.Printf("Error parsing configuration file %s: %v\n", obj.Key, err)
		return nil
	}

	if len(*parsed) == 0 {
		return nil
	}

	config := make([]*customizationConfig, 0, len(*parsed))
	for k, v := range *parsed {
		if k == "folder" {
			v.IsDefaultFolder = true
		} else if k == "file" {
			v.IsDefaultFile = true
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

func (dir *Directory) Render(semSort bool) {

	defaultDirConfig := &customizationEntry{Icon: "folder"}
	defaultFileConfig := &customizationEntry{Icon: "file"}
	for _, config := range dir.renderCustomization {
		if config.IsDefaultFolder && config.Default != nil {
			defaultDirConfig = config.Default
		} else if config.IsDefaultFile && config.Default != nil {
			defaultFileConfig = config.Default
		}
	}

	if len(dir.Folders) > 0 {
		dir.RenderedDirs = make([]*RenderizableDir, len(dir.Folders))
		for i, name := range dir.Folders {
			rd := new(RenderizableDir)
			rd.Name = name
			rd.Icon = defaultDirConfig.Icon
			rd.Description = defaultDirConfig.Description
			rd.version, _ = semver.NewVersion(name)
			dir.RenderedDirs[i] = rd
		}

		if semSort {
			sort.Sort(dirSemverCollection(dir.RenderedDirs))
		} else {
			sort.Sort(dirCollection(dir.RenderedDirs))
		}

		latest := make([]bool, len(dir.renderCustomization))

		for _, rd := range dir.RenderedDirs {
			for i, cust := range dir.renderCustomization {
				if cust.Pattern != nil && cust.Pattern.MatchString(rd.Name) {
					entry := cust.Default
					// The first entry that shows up for each pattern
					// is considered "latest"
					if !latest[i] {
						latest[i] = true
						if cust.Latest != nil {
							entry = cust.Latest
						}
					}
					rd.Description = cust.Pattern.ReplaceAllString(rd.Name, entry.Description)
					rd.Icon = entry.Icon
					break
				}
			}
		}
	}

	if len(dir.Files) > 0 {
		dir.RenderedFiles = make([]*RenderizableFile, len(dir.Files))
		i := 0
		for name, details := range dir.Files {
			rd := new(RenderizableFile)
			rd.Name = name
			rd.Icon = defaultFileConfig.Icon
			rd.Description = defaultFileConfig.Description
			rd.Bytes = details.Bytes
			rd.Date = details.Date
			dir.RenderedFiles[i] = rd
			i++
		}
		sort.Sort(fileCollection(dir.RenderedFiles))
	}
}

func (c fileCollection) Len() int {
	return len(c)
}

func (c fileCollection) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

func (c fileCollection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c dirCollection) Len() int {
	return len(c)
}

func (c dirCollection) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

func (c dirCollection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c dirSemverCollection) Len() int {
	return len(c)
}

func (c dirSemverCollection) Less(i, j int) bool {
	// Items with semver go first, ordered by version descending
	// The rest of the items go last, ordered alphabetically ascending
	if c[i].version == nil && c[j].version == nil {
		return c[i].Name < c[j].Name
	}
	if c[i].version != nil && c[j].version == nil {
		return true
	}
	if c[i].version == nil && c[j].version != nil {
		return false
	}
	// Note: this function sorts semver backwards;
	// we invert j with i
	// FIXME: 1.12.0-rc1 shows _after_ 1.12 instead of before
	if c[i].version.Equal(c[j].version) {
		// 1.1 is less than 1.1.0
		mi := semVerRegex.FindStringSubmatch(c[i].version.Original())
		mj := semVerRegex.FindStringSubmatch(c[j].version.Original())
		if mi != nil && mj != nil {
			return len(mj[0]) < len(mi[0])
		}
	}
	return c[j].version.LessThan(c[i].version)
}

func (c dirSemverCollection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
