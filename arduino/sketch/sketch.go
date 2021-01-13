// This file is part of arduino-cli.
//
// Copyright 2020 ARDUINO SA (http://www.arduino.cc/)
//
// This software is released under the GNU General Public License version 3,
// which covers the main part of arduino-cli.
// The terms of this license can be found at:
// https://www.gnu.org/licenses/gpl-3.0.en.html
//
// You can be released from the requirements of the above licenses by purchasing
// a commercial license. Buying such a license is mandatory if you want to
// modify or otherwise use the software for commercial activities involving the
// Arduino software without disclosing the source code of your own applications.
// To purchase a commercial license, send an email to license@arduino.cc.

package sketch

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arduino/arduino-cli/arduino/globals"
	"github.com/arduino/go-paths-helper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Item holds the source and the path for a single sketch file
type Item struct {
	Path string
}

// NewItem reads the source code for a sketch item and returns an
// Item instance
func NewItem(itemPath string) *Item {
	return &Item{itemPath}
}

// GetSourceBytes reads the item file contents and returns it as bytes
func (i *Item) GetSourceBytes() ([]byte, error) {
	// read the file
	source, err := ioutil.ReadFile(i.Path)
	if err != nil {
		return nil, errors.Wrap(err, "error reading source file")
	}
	return source, nil
}

// GetSourceStr reads the item file contents and returns it as a string
func (i *Item) GetSourceStr() (string, error) {
	source, err := i.GetSourceBytes()
	if err != nil {
		return "", err
	}
	return string(source), nil
}

// ItemByPath implements sort.Interface for []Item based on
// lexicographic order of the path string.
type ItemByPath []*Item

func (ibn ItemByPath) Len() int           { return len(ibn) }
func (ibn ItemByPath) Swap(i, j int)      { ibn[i], ibn[j] = ibn[j], ibn[i] }
func (ibn ItemByPath) Less(i, j int) bool { return ibn[i].Path < ibn[j].Path }

// Sketch holds all the files composing a sketch
type Sketch struct {
	MainFile         *Item
	LocationPath     string
	OtherSketchFiles []*Item
	AdditionalFiles  []*Item
}

// New creates an Sketch instance by reading all the files composing a sketch and grouping them
// by file type.
func New(sketchFolderPath, mainFilePath, buildPath string, allFilesPaths []string) (*Sketch, error) {
	var mainFile *Item

	// read all the sketch contents and create sketch Items
	pathToItem := make(map[string]*Item)
	for _, p := range allFilesPaths {
		// create an Item
		item := NewItem(p)

		if p == mainFilePath {
			// store the main sketch file
			mainFile = item
		} else {
			// map the file path to sketch.Item
			pathToItem[p] = item
		}
	}

	// organize the Items
	additionalFiles := []*Item{}
	otherSketchFiles := []*Item{}
	for p, item := range pathToItem {
		ext := strings.ToLower(filepath.Ext(p))
		if _, found := globals.MainFileValidExtensions[ext]; found {
			// item is a valid main file, see if it's stored at the
			// sketch root and ignore if it's not.
			if filepath.Dir(p) == sketchFolderPath {
				otherSketchFiles = append(otherSketchFiles, item)
			}
		} else if _, found := globals.AdditionalFileValidExtensions[ext]; found {
			// item is a valid sketch file, grab it only if the buildPath is empty
			// or the file is within the buildPath
			if buildPath == "" || !strings.Contains(filepath.Dir(p), buildPath) {
				additionalFiles = append(additionalFiles, item)
			}
		} else {
			return nil, errors.Errorf("unknown sketch file extension '%s'", ext)
		}
	}

	sort.Sort(ItemByPath(additionalFiles))
	sort.Sort(ItemByPath(otherSketchFiles))

	return &Sketch{
		MainFile:         mainFile,
		LocationPath:     sketchFolderPath,
		OtherSketchFiles: otherSketchFiles,
		AdditionalFiles:  additionalFiles,
	}, nil
}

// InitPath returns the current working directory, if sketchPath is not nil that is returned instead
func InitPath(sketchPath *paths.Path) (*paths.Path, error) {
	if sketchPath != nil {
		return sketchPath, nil
	}

	wd, err := paths.Getwd()
	if err != nil {
		return nil, fmt.Errorf("Couldn't get current working directory: %w", err)
	}
	logrus.Infof("Reading sketch from dir: %s", wd)
	return wd, nil
}
