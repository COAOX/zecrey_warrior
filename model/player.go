package model

import (
	"encoding/binary"
	"math"

	"github.com/solarlune/resolv"
)

const (
	PlayerTag           = "Player"
	defaultPlayerPixelR = minCellSize
)

type Player struct {
	ID   uint64 `json:"player_id"`
	Camp Camp   `json:"camp"`

	R int `json:"r"`

	Vx float64 `json:"vx"`
	Vy float64 `json:"vy"`

	playerObj *resolv.Object
}

func (p *Player) Serialize() []byte {
	b := make([]byte, 18)
	binary.LittleEndian.PutUint64(b[0:8], p.ID)
	b[8] = byte(p.Camp)
	b[9] = byte(p.R)
	if p.playerObj != nil {
		binary.LittleEndian.PutUint32(b[10:14], uint32(p.playerObj.X))
		binary.LittleEndian.PutUint32(b[14:18], uint32(p.playerObj.Y))
	}
	return b
}

func (p *Player) Size() uint32 {
	return 18
}

func (player *Player) rebound(dx, dy, rx, ry float64, cell *resolv.Object) (float64, float64) {
	// Edge Collision
	nx, ny := player.playerObj.X+dx, player.playerObj.Y+dy
	rx -= dx
	ry -= dy
	if nx >= cell.X && nx <= cell.X+cell.W {
		player.Vy = -player.Vy
		return rx, -ry
	}
	if ny >= cell.Y && ny <= cell.Y+cell.H {
		player.Vx = -player.Vx
		return -rx, ry
	}

	// Corner Collision
	remianV := math.Sqrt(rx*rx + ry*ry)
	if remianV == 0 {
		return 0, 0
	}

	v := math.Sqrt(player.Vx*player.Vx + player.Vy*player.Vy)
	player.Vx, player.Vy = v/float64(player.R)*(nx-cell.X), v/float64(player.R)*(ny-cell.Y)

	px, py := (nx - cell.X), (ny - cell.Y)
	pl := math.Sqrt(px*px + py*py)
	return remianV / pl * px, remianV / pl * py
}
