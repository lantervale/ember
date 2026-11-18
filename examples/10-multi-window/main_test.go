package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/camera"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/ui"
)

func TestEachWindowHasItsCameraAndUI(t *testing.T) {
	sceneWindow, mapWindow := newSceneWindow(), newMapWindow()
	w := ember.New().AddEntities(sceneWindow, mapWindow)

	var orbit *camera.OrbitCamera
	var eye *Eye
	var label *ui.Document
	for _, e := range w.Entities() {
		switch e := e.(type) {
		case *camera.OrbitCamera:
			orbit = e
		case *Eye:
			eye = e
		case *ui.Document:
			if e.Body().Text() == "map" {
				label = e
			}
		}
	}
	if orbit == nil || eye == nil || label == nil {
		t.Fatal("each window brings its camera and its label")
	}

	w.RunFrames(3)
	if orbit.RendersIn() != sceneWindow || eye.RendersIn() != mapWindow {
		t.Fatal("a camera draws in the window it is a child of")
	}
	if draw.FrontFor(w, nil) == nil || draw.FrontFor(w, mapWindow) == nil {
		t.Fatal("both windows get a UI")
	}

	w.RemoveEntity(mapWindow)
	if w.Has(eye) || w.Has(label) {
		t.Fatal("closing the map window takes its camera and label with it")
	}
}

func TestTheProgramEndsWithItsLastWindow(t *testing.T) {
	sceneWindow, mapWindow := newSceneWindow(), newMapWindow()
	w := ember.New().AddEntities(sceneWindow, mapWindow)
	w.RunFrames(2)

	sceneWindow.RequestClose()
	w.RunFrames(2)
	if w.Has(sceneWindow) || !w.Has(mapWindow) || w.ShouldQuit() {
		t.Fatal("closing the scene window leaves the map open and the program running")
	}
	mapWindow.RequestClose()
	w.RunFrames(1)
	if !w.ShouldQuit() {
		t.Fatal("closing the last window ends the program")
	}
}
