package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/solarlune/resolv"
)

const (
	EdgeTag           = "EDGE"
	HorizontalEdgeTag = "HORIZONTAL"
	VerticalEdgeTag   = "VERTICAL"

	minCellSize = 5
	edgeWidth   = defaultPlayerPixelR

	playerInitialVelocity = 2
)

type Game struct {
	Map     Map      `json:"map"`
	Players sync.Map `json:"players"`

	space *resolv.Space

	frameNumber uint32
}

func NewGame() *Game {
	v := &Game{
		Map:     NewMap(),
		Players: sync.Map{},
	}

	v.space = resolv.NewSpace(int(v.Map.Column*v.Map.CellWidth)+2*edgeWidth, int(v.Map.Row*v.Map.CellHeight)+2*edgeWidth, minCellSize, minCellSize)
	v.space.Add(resolv.NewObject(0, 0, float64(v.Map.Column*v.Map.CellWidth+edgeWidth), edgeWidth, EdgeTag, HorizontalEdgeTag))
	v.space.Add(resolv.NewObject(0, 0, edgeWidth, float64(v.Map.Column*v.Map.CellWidth+edgeWidth), EdgeTag, VerticalEdgeTag))
	v.space.Add(resolv.NewObject(float64(v.Map.Column*v.Map.CellWidth+edgeWidth), 0, edgeWidth, float64(v.Map.Row*v.Map.CellHeight+edgeWidth), EdgeTag, VerticalEdgeTag))
	v.space.Add(resolv.NewObject(0, float64(v.Map.Row*v.Map.CellHeight+edgeWidth), float64(v.Map.Column*v.Map.CellWidth+edgeWidth), edgeWidth, EdgeTag, HorizontalEdgeTag))

	for i := 0; i < int(v.Map.Row); i++ {
		for j := 0; j < int(v.Map.Column); j++ {
			camp := initCamp(i, j, int(v.Map.Row), int(v.Map.Column))
			v.space.Add(resolv.NewObject(float64(j*int(v.Map.CellWidth+edgeWidth)), float64(i*int(v.Map.CellHeight+edgeWidth)), float64(v.Map.CellWidth), float64(v.Map.CellHeight), CampTagMap[camp], CellIndexToTag(j, i)))
			v.Map.Cells = append(v.Map.Cells, camp)
		}
	}

	//TODO
	v.AddPlayer(1231231, ETH)
	v.AddPlayer(1211111, BNB)
	v.AddPlayer(11111, BTC)
	v.AddPlayer(111, AVAX)
	v.AddPlayer(222222, MATIC)

	return v
}

// frame number: 4 bytes
// map size: 4 bytes
// map: map size bytes
// players: 18 * len(players) bytes
func (g *Game) Serialize() ([]byte, error) {
	atomic.AddUint32(&g.frameNumber, 1)
	bytesBuf := bytes.NewBuffer([]byte{})
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, g.Size())
	_, err := bytesBuf.Write(b)
	if err != nil {
		return bytesBuf.Bytes(), err
	}
	binary.LittleEndian.PutUint32(b, g.frameNumber)
	_, err = bytesBuf.Write(b)
	if err != nil {
		return bytesBuf.Bytes(), err
	}

	binary.LittleEndian.PutUint32(b, g.Map.Size())
	bytesBuf.Write(b)
	bytesBuf.Write(g.Map.Serialize())

	g.Players.Range(func(key, value interface{}) bool {
		if v, ok := value.(*Player); ok && v != nil {
			bytesBuf.Write(v.Serialize())
		}
		return true
	})

	// by, _ := json.Marshal(g)
	// fmt.Println("game", string(by))
	// fmt.Println("cells", g.Map.Cells)

	return bytesBuf.Bytes(), nil
}

func (g *Game) Update() {
	g.Players.Range(func(key, value interface{}) bool {
		if player, ok := value.(*Player); ok && player != nil && player.playerObj != nil {
			remainX, remainY := player.Vx, player.Vy
			// fmt.Println("camp:", player.Camp, "x:", player.playerObj.X, "y:", player.playerObj.Y, "vx:", player.Vx, "vy:", player.Vy)
			// if player.playerObj.X < edgeWidth || player.playerObj.Y < edgeWidth || player.playerObj.X > g.Map.GetMapWidth()-edgeWidth || player.playerObj.Y > g.Map.GetMapHeight()-edgeWidth {
			// 	panic(fmt.Sprintln("camp:", player.Camp, "x:", player.playerObj.X, "y:", player.playerObj.Y, "vx:", player.Vx, "vy:", player.Vy))
			// }
			for remainX != 0 || remainY != 0 {
				dx, dy := remainX, remainY
				// fmt.Println("dx", dx, "dy", dy)
				if collision := player.playerObj.Check(dx, dy, getCollisionTags(player.Camp)...); collision != nil {
					// fmt.Println("##collision", collision)
					collisionObj := collision.Objects[0]
					dx = collision.ContactWithObject(collisionObj).X()
					dy = collision.ContactWithObject(collisionObj).Y()
					// fmt.Println("collision dx", dx, "collision dy", dy)
					if !collisionObj.HasTags(EdgeTag) {
						remainX, remainY = player.rebound(dx, dy, remainX, remainY, collisionObj)
						x, y := GetCellIndex(collisionObj.Tags())
						g.Map.Cells[y*int(g.Map.Column)+x] = player.Camp
						collisionObj.RemoveTags(removeCampTags(collisionObj.Tags())...)
						collisionObj.AddTags(CampTagMap[player.Camp])
					} else if collisionObj.HasTags(HorizontalEdgeTag) {
						player.Vy = -player.Vy
						remainX -= dx
						remainY = dy - remainY
					} else {
						player.Vx = -player.Vx
						remainX = dx - remainX
						remainY -= dy
					}
				} else {
					remainX -= dx
					remainY -= dy
				}
				// fmt.Println("#inner camp:", player.Camp, "x:", player.playerObj.X, "y:", player.playerObj.Y, "dx:", dx, "dy:", dy, "vx:", player.Vx, "vy:", player.Vy, "rx:", remainX, "ry:", remainY)
				player.playerObj.X += dx
				player.playerObj.Y += dy
				player.playerObj.Update()
			}
			g.Players.Store(key, player)
		}
		return true
	})
}

func (g *Game) Size() uint32 {
	pLen := uint32(0)
	g.Players.Range(func(key, value interface{}) bool { // O(N) call, but since players are not that many, it's fine
		if v, ok := value.(*Player); ok && v != nil {
			pLen += v.Size()
		}
		return true
	})
	return 4 + 4 + g.Map.Size() + pLen
}

func (g *Game) AddPlayer(playerID uint64, camp Camp) *Player {
	x, y := camp.Center(int(g.Map.Row), int(g.Map.Column)) // cell index
	x *= int(g.Map.CellWidth)                              // pixel index
	y *= int(g.Map.CellHeight)

	ang := rand.Float64() * 2 * math.Pi
	player := &Player{
		ID:   playerID,
		Camp: camp,
		R:    defaultPlayerPixelR,
		Vx:   math.Cos(ang) * playerInitialVelocity,
		Vy:   math.Sin(ang) * playerInitialVelocity,
	}
	player.playerObj = resolv.NewObject(float64(x-player.R+edgeWidth), float64(y-player.R+edgeWidth), float64(2*player.R), float64(2*player.R), PlayerTag)
	player.playerObj.SetShape(resolv.NewCircle(float64(player.R), float64(player.R), float64(player.R)))
	g.space.Add(player.playerObj)
	g.Players.Store(playerID, player)

	// fmt.Println("new player, camp:", camp, "x:", player.playerObj.X, "y:", player.playerObj.Y, "vx:", player.Vx, "vy:", player.Vy)
	return player
}

func GetCellIndex(tags []string) (int, int) {
	for _, tag := range tags {
		s := strings.Split(tag, ",")
		if len(s) == 2 {
			x, _ := strconv.Atoi(s[0])
			y, _ := strconv.Atoi(s[1])
			return x, y
		}
	}
	return 0, 0
}

func CellIndexToTag(x, y int) string {
	return fmt.Sprintf("%d,%d", x, y)
}

func CellTagToIndex(tag string) (int, int) {
	s := strings.Split(tag, ",")
	y, _ := strconv.Atoi(s[0])
	x, _ := strconv.Atoi(s[1])
	return x, y
}
