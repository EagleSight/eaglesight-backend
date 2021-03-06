package world

import (
	"log"
	"testing"
	"time"

	"github.com/eaglesight/eaglesight-server/mathutils"
)

func getTestWorld() *World {
	terrain, err := LoadTerrain("../map.esmap")

	if err != nil {
		log.Fatalln(err)
	}

	return NewWorld(terrain)
}

func TestNewWorld(t *testing.T) {

	w := getTestWorld()
	//log.Print("world created")
	go func(world *World) {
		time.Sleep(time.Second)
		//log.Print("Should end now")
		w.End <- false
	}(w)

	go func(world *World) {
		for {
			<-w.Snapshots
		}
	}(w)

	w.Run(time.Second/100, time.Second/20)
}

func TestAddBullet(t *testing.T) {

	w := getTestWorld()

	b := Bullet{}

	w.addBullet(&b)

	if len(w.bullets) == 0 {
		t.Fail()
	}
}

func TestAddPlane(t *testing.T) {
	w := getTestWorld()

	w.addPlane(1, PlaneModel{}, w.gun)

	if len(w.planes) == 0 {
		t.Fail()
	}

	w.applyInput(&PlayerInput{UID: 1, Data: []byte{0x3, 12, 12, 12, 26, 0x80}})

	if !w.planes[1].input.IsFiring {
		t.Fail()
	}
}

func BenchmarkXPlayers(b *testing.B) {
	w := getTestWorld()
	const X = 1

	for i := 1; i <= X; i++ {
		w.addPlane(uint8(i), PlaneModel{}, w.gun)
	}
	deltaT := float64(time.Second / 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.updateWorld(deltaT)
		if i%5 == 0 {
			w.generateSnapshots()
		}
	}
}

func BenchmarkBullet(b *testing.B) {
	m := mathutils.NewMatrix3()
	bullet := NewBullet(2, mathutils.Vector3D{X: 0, Y: 0, Z: 0}, &m, 600, 5)

	deltaT := float64(time.Second / 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bullet.Update(deltaT)
	}
}
