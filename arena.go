package main

import (
	"encoding/binary"
	"math"
	"sync"
	"time"
)

const (
	// UpdateDataLenght is uint32-uid|3*float32-position|3*float32-location
	UpdateDataLenght = 7 * 4
)

// Arena riens
type Arena struct {
	players        map[*Player]bool
	connect        chan *Player
	deconect       chan *Player
	input          chan *PlayerInput
	snapshotInputs map[uint32]*PlayerInput
	tick           uint32
	mux            sync.Mutex
	arenaMap       *ArenaMap
}

// NewArena return a arena with default settings
func NewArena() *Arena {

	arenaMap := LoadArenaMap()

	return &Arena{
		players:        make(map[*Player]bool),
		connect:        make(chan *Player),
		deconect:       make(chan *Player),
		input:          make(chan *PlayerInput),
		snapshotInputs: make(map[uint32]*PlayerInput),
		tick:           0,
		arenaMap:       &arenaMap,
	}
}

func generateSnapshot(a *Arena, deltaT float64) []byte {

	defer a.mux.Unlock()

	// We lock the mutex because we want to make sure that nobody else append a state while the inputsPacket is made
	a.mux.Lock()

	a.tick++

	offset := 1 + 4 + 2 // unt8 + uint32 + uint16
	const playerDataLenght = 28
	snapshot := make([]byte, offset+len(a.snapshotInputs)*playerDataLenght)

	snapshot[0] = uint8(3)
	binary.BigEndian.PutUint32(snapshot[1:], uint32(a.tick))
	binary.BigEndian.PutUint16(snapshot[5:], uint16(len(a.snapshotInputs)))

	for k, v := range a.snapshotInputs {

		v.plane.Update(v.data, deltaT)

		// Dump everything into the slice
		binary.BigEndian.PutUint32(snapshot[offset:], v.plane.uid)

		// Location
		binary.BigEndian.PutUint32(snapshot[offset+4:], math.Float32bits(float32(v.plane.location.x)))
		binary.BigEndian.PutUint32(snapshot[offset+8:], math.Float32bits(float32(v.plane.location.y)))
		binary.BigEndian.PutUint32(snapshot[offset+12:], math.Float32bits(float32(v.plane.location.z)))

		// Rotation
		binary.BigEndian.PutUint32(snapshot[offset+16:], math.Float32bits(float32(v.plane.absRot.x)))
		binary.BigEndian.PutUint32(snapshot[offset+20:], math.Float32bits(float32(v.plane.absRot.y)))
		binary.BigEndian.PutUint32(snapshot[offset+24:], math.Float32bits(float32(v.plane.absRot.z)))

		offset += playerDataLenght

		a.snapshotInputs[k] = &PlayerInput{plane: v.plane, data: nil}

	}

	return snapshot

}

func (a *Arena) broadcastSnapshots() {

	previousTickTime := time.Now()

	c := time.Tick(time.Second / 60)

	for now := range c {

		// Calculate the time since the last time updated
		deltaT := now.Sub(previousTickTime).Seconds()

		// Save for the next time
		previousTickTime = now

		// Send inputs to all the players
		a.Broadcast(generateSnapshot(a, deltaT))

	}
}

// Run start the Arena
func (a *Arena) Run() {

	go a.broadcastSnapshots()

	for {
		select {
		case player := <-a.connect:
			a.connectPlayer(player)
		case player := <-a.deconect:
			a.deconnectPlayer(player)
		case input := <-a.input:
			a.mux.Lock()
			a.snapshotInputs[input.plane.uid] = input
			a.mux.Unlock()
		}
	}
}

// Broadcast sent a []byte containing a payload to all the players
func (a *Arena) Broadcast(payload []byte) {

	// Send the payload to all the players
	for p := range a.players {
		p.send <- payload
	}

}

func (a *Arena) connectPlayer(player *Player) {

	player.sendPlayersList()

	a.players[player] = true

	a.snapshotInputs[player.uid] = &PlayerInput{plane: player.plane, data: nil}

	// 0x1 - player's uid ----
	message := make([]byte, 5)

	message[0] = 0x1 // Connection
	binary.BigEndian.PutUint32(message[1:], player.uid)

	go a.Broadcast(message)
}

func (a *Arena) deconnectPlayer(player *Player) {

	a.mux.Lock()

	// Remove the player from the players list
	if _, ok := a.players[player]; ok {
		delete(a.players, player)
		delete(a.snapshotInputs, player.uid)

		message := make([]byte, 5)

		message[0] = 0x2 // Deconnection
		binary.BigEndian.PutUint32(message[1:], player.uid)

		go a.Broadcast(message)
	}

	a.mux.Unlock()
}
