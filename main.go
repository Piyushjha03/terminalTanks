package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var gravity = 9.81

type model struct {
	terrain    []int
	tankPos    int
	targetPos  int
	ballPosX   int
	ballPosY   int
	angle      float64
	power      float64
	simulating bool
	hit        bool
	timePassed float64
}

var terrainStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
var tankStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
var targetStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
var ballStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
var borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Bold(true)

// generateTerrain generates a more realistic terrain using linear interpolation and superposition
func generateTerrain(width int, iterations int) []int {
	naiveTerrain := make([]float64, width)
	for i := range naiveTerrain {
		naiveTerrain[i] = float64(rand.Intn(10) + 5)
	}

	var terrains [][]float64
	weightSum := 0.0

	for z := iterations; z > 0; z-- {
		terrain := make([]float64, 0, width)
		weight := 1 / math.Pow(2, float64(z-1))
		sample := 1 << (iterations - z)

		samplePoints := make([]float64, 0)
		for i := 0; i < len(naiveTerrain); i += sample {
			samplePoints = append(samplePoints, naiveTerrain[i])
		}

		weightSum += weight

		for i := 0; i < len(samplePoints); i++ {
			terrain = append(terrain, weight*samplePoints[i])
			for j := 1; j < sample; j++ {
				mu := float64(j) / float64(sample)
				a := samplePoints[i]
				b := samplePoints[(i+1)%len(samplePoints)]
				v := cosineInterpolation(a, b, mu)
				terrain = append(terrain, weight*v)
			}
		}
		terrains = append(terrains, terrain)
	}

	finalTerrain := make([]float64, len(naiveTerrain))
	for i := range finalTerrain {
		for _, t := range terrains {
			if i < len(t) {
				finalTerrain[i] += t[i]
			}
		}
		finalTerrain[i] /= weightSum
	}

	// Convert terrain heights to integers for simplicity
	terrain := make([]int, len(finalTerrain))
	for i := range finalTerrain {
		terrain[i] = int(math.Round(finalTerrain[i]))
	}
	return terrain
}

func cosineInterpolation(a, b, mu float64) float64 {
	mu2 := (1 - math.Cos(mu*math.Pi)) / 2
	return a*(1-mu2) + b*mu2
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
		if !m.simulating {
			switch msg.String() {
			case "a":
				m.angle -= 5
			case "d":
				m.angle += 5
			case "w":
				m.power += 1
			case "s":
				m.power -= 1
			case "enter":
				m.simulating = true
				m.timePassed = 0
				return m, tick()
			}
		}
	case tickMsg:
		return m.simulate()
	case resetMsg:
		return m.resetGame(), nil
	}
	return m, nil
}

func (m model) View() string {
	view := borderStyle.Render(fmt.Sprintf("Angle: %.1f° | Power: %.1f | Press 'q' to quit", m.angle, m.power)) + "\n"
	if m.hit {
		view += "🎯 You hit the target! Press 'q' to quit.\n"
	} else if !m.simulating && !m.hit {
		view += "❌ Missed! Game restarting...\n"
	}
	view += displayTerrainWithTank(m.terrain, m.tankPos, m.targetPos, m.ballPosX, m.ballPosY)
	return view
}

func displayTerrainWithTank(terrain []int, tankPos, targetPos, ballPosX, ballPosY int) string {
	view := ""
	for y := 30; y >= 0; y-- {
		for x, h := range terrain {
			switch {
			case x == ballPosX && y == ballPosY:
				view += ballStyle.Render("O")
			case x == tankPos && h == y:
				view += tankStyle.Render("T")
			case x == targetPos && h == y:
				view += targetStyle.Render("X")
			case h >= y:
				view += terrainStyle.Render("|")
			default:
				view += " "
			}
		}
		view += "\n"
	}
	return view
}

type tickMsg struct{}
type resetMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func reset() tea.Cmd {
	return tea.Tick(time.Second*2, func(_ time.Time) tea.Msg {
		return resetMsg{}
	})
}

func (m model) simulate() (tea.Model, tea.Cmd) {
	if !m.simulating {
		return m, nil
	}

	angleRad := m.angle * math.Pi / 180
	xPos := float64(m.tankPos)
	yPos := float64(m.terrain[m.tankPos])

	m.timePassed += 0.1
	newXPos := xPos + m.power*math.Cos(angleRad)*m.timePassed
	newYPos := yPos + m.power*math.Sin(angleRad)*m.timePassed - 0.5*gravity*m.timePassed*m.timePassed

	if int(newXPos) >= len(m.terrain) || int(newXPos) < 0 || newYPos <= float64(m.terrain[int(newXPos)]) {
		m.simulating = false
		return m, reset()
	}

	m.ballPosX = int(math.Round(newXPos))
	m.ballPosY = int(math.Round(newYPos))

	if int(math.Abs(float64(m.targetPos-m.ballPosX))) <= 3 &&
		m.ballPosY >= m.terrain[m.targetPos]-2 && m.ballPosY <= m.terrain[m.targetPos]+2 {
		m.hit = true
		m.simulating = false
		return m, nil
	}

	return m, tick()
}

func (m model) resetGame() model {
	return model{
		terrain:    generateTerrain(100, 6),
		tankPos:    rand.Intn(5) + 1,
		targetPos:  rand.Intn(5) + 75,
		ballPosX:   rand.Intn(5) + 1,
		ballPosY:   m.terrain[rand.Intn(5)+1],
		angle:      45,
		power:      20,
		simulating: false,
		hit:        false,
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	terrain := generateTerrain(100, 6)
	tankPos := rand.Intn(5) + 1
	targetPos := rand.Intn(5) + 75

	initialModel := model{
		terrain:    terrain,
		tankPos:    tankPos,
		targetPos:  targetPos,
		ballPosX:   tankPos,
		ballPosY:   terrain[tankPos],
		angle:      45,
		power:      20,
		simulating: false,
		hit:        false,
	}

	p := tea.NewProgram(initialModel)
	if err := p.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting game: %v", err)
		os.Exit(1)
	}
}
