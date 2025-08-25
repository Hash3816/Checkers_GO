package main

import (
	"encoding/json"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var russianFont, russianFont2 font.Face
var (
	rulesText = `Чтобы во время игры увидеть правила нажмите клавишу "Tab"
Чтобы выйти в главное меню нажмите клавишу "Esc"
Чтобы сохранить игру и выйти нажмите клавишу "S"
1. У каждого игрока по 12 шашек, которые расставляются по чёрным полям на
доске в 64 клетки (8х8).
2. Шашки двигаются только по чёрным полям, только вперёд по диагонали, на
свободное соседнее поле.
3. Начинают игру белые. Ходы делаются по очереди. Нельзя делать несколько ходов подряд.
4. Если белая и чёрная шашки находятся на соседних полях по диагонали, а поле
за шашкой противника свободно, то тот играющий, чья очередь ходить, обязан
побить (взять) шашку противника.
5. Брать шашку соперника можно ходом вперёд и ходом назад, если за ней есть
свободное поле.
6. Если шашка, двигаясь по доске, достигает переднего ряда, она сразу превращается в дамку,
которая может ходить, и вперёд, и назад по всей диагонали.
7. Если за шашкой соперника имеется несколько свободных полей подряд, дамка
может остановиться на любом из них.
8. Дамка может сразу брать несколько шашек, если за каждой из них есть
хотя бы одно свободное поле.
9. При обоюдном зажиме проигрывает тот, чья очередь хода.`
)

type Cell struct {
	Color_check string // white,black, clear - если нет ВАЖНО если clear на параметр is_king даже не смотреть
	Color_cell  string // white, black, green, red - green отрисовка что шашка может сюда сходить, red что шашка должна бить, white всегда пустые клетки между пешками
	Is_king     bool
}
type Cords struct {
	X int
	Y int
}
type Beat struct {
	Begin_pos      Cords
	Will_killed    []Cords
	New_pos        Cords
	Is_become_king bool
}
type check_and_kor struct {
	Pos     Cords
	Is_king bool // check, king
}
type Menu struct {
	StartButton    image.Rectangle
	ContinueButton image.Rectangle
	ExitButton     image.Rectangle
	RulesButton    image.Rectangle
	Active         bool
}
type Game_Buttons struct {
	Save_Button image.Rectangle
	Exit_Botton image.Rectangle
	Rule_Button image.Rectangle
	Tie_Button  image.Rectangle
	RegularFont font.Face
}
type MassageScreen struct {
	Active     bool
	Winner     string
	MenuButton image.Rectangle
}
type RulesScreen struct {
	Active      bool
	BackButton  image.Rectangle
	TextLines   []string
	ScrollY     float64
	TitleFont   font.Face
	RegularFont font.Face
}
type Game struct {
	Width, Height         int
	Field                 [8][8]Cell
	Now_player            string  // white, black
	Clicks                []Cords // Хранит координаты кликов
	Change_color          []Cords // Координаты которые были окрашены в зелёный или красный
	Need_beat             []Beat  // Список пешек которые нужно рубить на выбор
	Winner                string  // white, black, None
	LastClickTime         int64   // Время последнего клика в наносекундах
	ClickDelay            int64   // Задержка между кликами в наносекундах (например, 200ms)
	Menu                  Menu
	Buttons_in_game       Game_Buttons
	RulesScreen           RulesScreen
	MassageScreen         MassageScreen
	InMenu                bool
	ShowRulesInGame       bool
	InGame                bool
	Proccesed_Exit_Button bool
	Proccesed_Tie_Button  bool
}

func (game *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if !game.RulesScreen.Active {
			*game = *initGame() // Полностью перезагружаем игру
			return nil
		}
		if game.RulesScreen.Active {
			game.RulesScreen.Active = false
			game.Menu.Active = true
			return nil
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyY) && game.Proccesed_Exit_Button {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) && game.Proccesed_Exit_Button {
		game.Proccesed_Exit_Button = false
		return nil
	}
	if game.Proccesed_Exit_Button {
		return nil
	}

	if game.MassageScreen.Active { // Для того чтобы игрок ничего не тыкал если согаситься на ничю или произойдёт победа
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyY) && game.Proccesed_Tie_Button {
		game.Proccesed_Tie_Button = false
		game.MassageScreen.Active = true
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) && game.Proccesed_Tie_Button {
		game.Proccesed_Tie_Button = false
		return nil
	}
	if game.Proccesed_Tie_Button {
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyS) && game.InGame && !game.MassageScreen.Active { // Горячая клавиша сохранения
		json_game, err := json.MarshalIndent(game, "", "  ") // Форматируем с отступами
		if err != nil {
			panic(err)
		}
		err = os.WriteFile("save.json", json_game, 0644)
		if err != nil {
			panic(err)
		}
		game.Menu.Active = true
	}
	// Tab
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if !game.RulesScreen.Active && !game.Menu.Active && !game.MassageScreen.Active {
			game.ShowRulesInGame = !game.ShowRulesInGame
		}
		return nil
	}
	// Обработка кликов меню
	if game.Menu.Active {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			pos := image.Point{x, y}

			if pos.In(game.Menu.StartButton) {
				*game = *initGame() // Инициализация игры с начальной расстановкой
				game.Menu.Active = false
				game.InGame = true
			} else if pos.In(game.Menu.ContinueButton) { // Продолжить
				json_save, err := os.ReadFile("save.json")
				if err != nil {
					return nil
				}
				err = json.Unmarshal(json_save, &game)
				if err != nil {
					panic(err)
				}
				game.Menu.Active = false
				game.InGame = true
			} else if pos.In(game.Menu.RulesButton) { // Правила
				game.Menu.Active = false
				game.RulesScreen.Active = true
			} else if pos.In(game.Menu.ExitButton) { // Выход
				return ebiten.Termination
			}
		}
		return nil
	}

	if game.RulesScreen.Active {
		// Обработка кнопки "Назад"
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			pos := image.Point{x, y}

			if pos.In(game.RulesScreen.BackButton) {
				game.RulesScreen.Active = false
				game.Menu.Active = true
				game.RulesScreen.ScrollY = 0 // Сброс прокрутки при выходе
			}
		}
		return nil
	}

	//Если пешки у игрока кончились то устанавливаем победителя
	if len(give_positions_checks(game.Now_player, game.Field)) == 0 {
		game.Winner = change_player(game.Now_player)
		game.MassageScreen.Winner = game.Winner
		game.MassageScreen.Active = true
		return nil
	}
	if game.Now_player == "black" {
		game.moveAI()
		return nil
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) { // Обработка кликов мыши в игре
		if !game.canClick() {
			return nil
		}
		game.LastClickTime = time.Now().UnixNano()
		x_input, y_input := ebiten.CursorPosition()

		if y_input > 640 { // Область кнопок
			x, y := ebiten.CursorPosition()
			pos := image.Point{x, y}

			if pos.In(game.Buttons_in_game.Save_Button) { // Нажата кнопка сохранения
				json_game, err := json.MarshalIndent(game, "", "  ") // Форматируем с отступами
				if err != nil {
					panic(err)
				}
				err = os.WriteFile("save.json", json_game, 0644)
				return nil

			} else if pos.In(game.Buttons_in_game.Rule_Button) { //Нажата кнопка вывода правил
				if !game.RulesScreen.Active && !game.Menu.Active && !game.MassageScreen.Active {
					game.ShowRulesInGame = !game.ShowRulesInGame
				}
				return nil

			} else if pos.In(game.Buttons_in_game.Exit_Botton) { //Нажата кнопка выхода из игры
				game.Proccesed_Exit_Button = true
				return nil
			} else if pos.In(game.Buttons_in_game.Tie_Button) {
				game.Proccesed_Tie_Button = true
				return nil
			}

		}
		y := 7 - (y_input / cellSize)
		x := x_input / cellSize

		// Проверяем, что клик в пределах поля
		if (x >= 0 && x <= 7) && (y >= 0 && y <= 7) {
			if len(game.Clicks) == 0 && game.Field[x][y].Color_check == game.Now_player {
				game.Clicks = append(game.Clicks, Cords{x, y})
				game.move(game.Clicks, game.Field)
				return nil

			} else if len(game.Clicks) == 1 && game.Field[x][y].Color_cell != "white" && game.Field[x][y].Color_check == "clear" {
				game.Clicks = append(game.Clicks, Cords{x, y})
				game.move(game.Clicks, game.Field)
				for i := 0; i < len(game.Change_color); i++ {
					x_1, y_1 := game.Change_color[i].X, game.Change_color[i].Y
					game.Field[x_1][y_1].Color_cell = "black"
				}
				game.Change_color = game.Change_color[:0]
				game.Clicks = game.Clicks[:0]
				return nil

			} else {
				for i := 0; i < len(game.Change_color); i++ {
					x_1, y_1 := game.Change_color[i].X, game.Change_color[i].Y
					game.Field[x_1][y_1].Color_cell = "black"
				}
				game.Clicks = game.Clicks[:0]
				game.Change_color = game.Change_color[:0]
				return nil
			}
		} else {
			return nil
		}

	}
	return nil
}

func (game *Game) canClick() bool {
	now := time.Now().UnixNano()
	return now-game.LastClickTime >= game.ClickDelay
}

func (game *Game) move(clicks []Cords, field [8][8]Cell) { // Обработка ходов
	Player := game.Now_player

	need_beat := Get_beats(Player, field)

	if len(need_beat) > 0 {
		if len(clicks) == 1 {
			for i := 0; i < len(need_beat); i++ {
				if need_beat[i].Begin_pos == clicks[0] {
					x, y := need_beat[i].New_pos.X, need_beat[i].New_pos.Y
					game.Field[x][y].Color_cell = "red"
					game.Change_color = append(game.Change_color, Cords{x, y})
				}
			}
			return
		} else if len(clicks) == 2 {
			var spisok_killed []Cords
			Flag := false // Проверка на выбрал ли игрок клетку взятия
			for i := 0; i < len(need_beat); i++ {
				if need_beat[i].Begin_pos == clicks[0] && need_beat[i].New_pos == clicks[1] {
					x1, y1 := need_beat[i].Begin_pos.X, need_beat[i].Begin_pos.Y
					x2, y2 := need_beat[i].New_pos.X, need_beat[i].New_pos.Y
					game.Field[x1][y1].Color_check = "clear"
					game.Field[x2][y2].Color_check = Player
					game.Field[x2][y2].Is_king = need_beat[i].Is_become_king
					spisok_killed = need_beat[i].Will_killed
					Flag = true
					break
				}
			}
			if Flag == false {
				return
			}
			for j := 0; j < len(spisok_killed); j++ {
				x, y := spisok_killed[j].X, spisok_killed[j].Y
				game.Field[x][y].Color_check = "clear"
				game.Field[x][y].Is_king = false
			}
			game.Now_player = change_player(Player)
			return
		}
	}
	if len(need_beat) == 0 {
		x, y := clicks[0].X, clicks[0].Y
		if !Is_Can_move(give_positions_checks(Player, field), field, Player) {
			game.Winner = change_player(Player)
			game.MassageScreen.Winner = game.Winner
			game.MassageScreen.Active = true
			return
		}

		if field[x][y].Is_king == false { //Код для хода шашки 1 клик подсветка возможных ходов 2 клик ход если он возможен
			if len(clicks) == 1 {
				if Player == "black" {
					if x-1 >= 0 && y-1 >= 0 && game.Field[x-1][y-1].Color_check == "clear" {
						game.Field[x-1][y-1].Color_cell = "green"
						game.Change_color = append(game.Change_color, Cords{x - 1, y - 1})
					}
					if x+1 <= 7 && y-1 >= 0 && game.Field[x+1][y-1].Color_check == "clear" {
						game.Field[x+1][y-1].Color_cell = "green"
						game.Change_color = append(game.Change_color, Cords{x + 1, y - 1})
					}
					return
				} else if Player == "white" {
					if x-1 >= 0 && y+1 <= 7 && game.Field[x-1][y+1].Color_check == "clear" {
						game.Field[x-1][y+1].Color_cell = "green"
						game.Change_color = append(game.Change_color, Cords{x - 1, y + 1})
					}
					if x+1 <= 7 && y+1 <= 7 && game.Field[x+1][y+1].Color_check == "clear" {
						game.Field[x+1][y+1].Color_cell = "green"
						game.Change_color = append(game.Change_color, Cords{x + 1, y + 1})
					}
					return
				}

			} else if len(clicks) == 2 {
				x2, y2 := clicks[1].X, clicks[1].Y
				if Player == "white" {
					if (x-1 == x2 && y+1 == y2) || (x+1 == x2 && y+1 == y2) {
						game.Field[x][y].Color_check = "clear"
						game.Field[x2][y2].Is_king = false
						game.Field[x2][y2].Color_check = Player
						game.Field[x2][y2].Is_king = false

						if y2 == 7 {
							game.Field[x2][y2].Is_king = true
						}
						game.Now_player = change_player(Player)
						return
					} else {
						return
					}

				} else if Player == "black" {
					if (x-1 == x2 && y-1 == y2) || (x+1 == x2 && y-1 == y2) {
						game.Field[x][y].Color_check = "clear"
						game.Field[x2][y2].Is_king = false
						game.Field[x2][y2].Color_check = Player
						game.Field[x2][y2].Is_king = false
						if y2 == 0 {
							game.Field[x2][y2].Is_king = true
						}
						game.Now_player = change_player(Player)
						return
					} else {
						return
					}
				}
				return
			}
		} else if field[x][y].Is_king == true {
			if len(clicks) == 1 {
				kords_king := king_pos(x, y, field)
				for i := 0; i < len(kords_king); i++ {
					game.Field[kords_king[i].X][kords_king[i].Y].Color_cell = "green"
					game.Change_color = append(game.Change_color, kords_king[i])
				}
				return
			}
			if len(clicks) == 2 {
				x2, y2 := clicks[1].X, clicks[1].Y
				if field[x2][y2].Color_cell == "green" {
					game.Field[x][y].Color_check = "clear"
					game.Field[x][y].Is_king = false
					game.Field[x2][y2].Color_check = Player
					game.Field[x2][y2].Is_king = true
					game.Now_player = change_player(Player)
				}
				return

			}
		}
	}
	return
}

func (game *Game) moveAI() {
	player := "black"

	moves := gen_player_moves(player, game.Field)

	bestMove := moves[0]
	bestScore := -1 << 30
	alpha := -1 << 30
	beta := 1 << 30

	for _, m := range moves {
		nextField := simulate_move(m, player, game.Field)
		// глубина 3: сейчас сделали ход чёрных -> ход белых (min) -> ход чёрных (max) -> оценка
		score := minimaxAlphaBeta(nextField, 2, false, alpha, beta)
		if score > bestScore {
			bestScore = score
			bestMove = m
		}
		// Лёгкое «aspiration»: подвинем alpha вокруг лучшего
		if score > alpha {
			alpha = score
		}
	}

	// применяем лучший найденный ход
	game.Field = simulate_move(bestMove, player, game.Field)

	game.Now_player = change_player(player)
}

func evaluateBoard(field [8][8]Cell) int {
	score := 0
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			c := field[x][y]
			switch c.Color_check {
			case "black":
				if c.Is_king {
					score += 5
				} else {
					score += 3 + (7 - y)
				}
			case "white":
				if c.Is_king {
					score -= 5
				} else {
					score -= 3 + y
				}
			}
		}
	}
	return score
}

func gen_player_moves(player string, field [8][8]Cell) []Beat {
	var moves []Beat

	// 1) Сначала — обязательные взятия
	beats := Get_beats(player, field)
	if len(beats) > 0 {
		moves = make([]Beat, 0, len(beats))
		for _, b := range beats {
			moves = append(moves, Beat{
				Begin_pos:      b.Begin_pos,
				New_pos:        b.New_pos,
				Will_killed:    append([]Cords(nil), b.Will_killed...),
				Is_become_king: b.Is_become_king,
			})
		}
		return moves
	}

	positions := give_positions_checks(player, field)
	for _, p := range positions {
		x, y := p.Pos.X, p.Pos.Y
		if !p.Is_king {
			dy := -1
			if player == "white" {
				dy = 1
			}
			if x-1 >= 0 && y+dy >= 0 && y+dy < 8 && field[x-1][y+dy].Color_check == "clear" {
				moves = append(moves, Beat{
					Begin_pos: p.Pos, New_pos: Cords{X: x - 1, Y: y + dy},
				})
			}
			if x+1 < 8 && y+dy >= 0 && y+dy < 8 && field[x+1][y+dy].Color_check == "clear" {
				moves = append(moves, Beat{
					Begin_pos: p.Pos, New_pos: Cords{X: x + 1, Y: y + dy},
				})
			}
		} else {
			opts := king_pos(x, y, field)
			for _, o := range opts {
				moves = append(moves, Beat{
					Begin_pos: p.Pos, New_pos: o, Is_become_king: true, // дамка остаётся дамкой
				})
			}
		}
	}
	return moves
}

func simulate_move(move Beat, player string, field [8][8]Cell) [8][8]Cell {
	newField := field
	bx, by := move.Begin_pos.X, move.Begin_pos.Y
	nx, ny := move.New_pos.X, move.New_pos.Y

	wasKing := newField[bx][by].Is_king

	newField[bx][by].Color_check = "clear"
	newField[bx][by].Is_king = false

	newField[nx][ny].Color_check = player

	becomeByEdge := false
	if !wasKing && len(move.Will_killed) == 0 { // обычный ход пешкой
		if player == "black" && ny == 0 {
			becomeByEdge = true
		}
		if player == "white" && ny == 7 {
			becomeByEdge = true
		}
	}
	newField[nx][ny].Is_king = wasKing || move.Is_become_king || becomeByEdge

	// снимаем побитых
	for _, c := range move.Will_killed {
		newField[c.X][c.Y].Color_check = "clear"
		newField[c.X][c.Y].Is_king = false
	}
	return newField
}

func minimaxAlphaBeta(field [8][8]Cell, depth int, maximizing bool, alpha, beta int) int {
	if depth == 0 {
		return evaluateBoard(field)
	}

	player := "black"
	if !maximizing {
		player = "white"
	}
	moves := gen_player_moves(player, field)
	if len(moves) == 0 {
		// Нет ходов — статическая оценка позиции (мог быть цугцванг/материал)
		return evaluateBoard(field)
	}

	if maximizing {
		best := -1 << 30
		for _, m := range moves {
			nextField := simulate_move(m, player, field)
			score := minimaxAlphaBeta(nextField, depth-1, false, alpha, beta)
			if score > best {
				best = score
			}
			if best > alpha {
				alpha = best
			}
			if beta <= alpha {
				break // отсечение
			}
		}
		return best
	} else {
		best := 1 << 30
		for _, m := range moves {
			nextField := simulate_move(m, player, field)
			score := minimaxAlphaBeta(nextField, depth-1, true, alpha, beta)
			if score < best {
				best = score
			}
			if best < beta {
				beta = best
			}
			if beta <= alpha {
				break // отсечение
			}
		}
		return best
	}
}

func give_positions_checks(Player string, field [8][8]Cell) (positions []check_and_kor) {
	var positions_checks []check_and_kor
	if Player == "black" {
		for y := 0; y < 8; y++ {
			x := 0

			if y%2 != 0 {
				x += 1
			}
			for x < 8 {
				if field[x][y].Color_check == "black" {
					var info check_and_kor
					info.Pos = Cords{x, y}
					info.Is_king = field[x][y].Is_king
					positions_checks = append(positions_checks, info)
				}
				x += 2
			}
		}

	} else if Player == "white" {
		for y := 0; y < 8; y++ {
			x := 0
			if y%2 != 0 {
				x += 1
			}
			for x < 8 {
				if field[x][y].Color_check == "white" {
					var info check_and_kor
					info.Pos = Cords{x, y}
					info.Is_king = field[x][y].Is_king
					positions_checks = append(positions_checks, info)
				}
				x += 2
			}
		}
	}
	return positions_checks
}

func get_checks_beats(pos Cords, player string, field [8][8]Cell, visited map[Cords]bool) []Beat {
	var beats []Beat                                          // Массив где храняться взятия(уже с комбинациями если они есть)
	directions := []Cords{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} // Возможные ходы

	for _, dir := range directions {
		enemyPos := Cords{pos.X + dir.X, pos.Y + dir.Y}   //Возможные позиции врага
		newPos := Cords{pos.X + 2*dir.X, pos.Y + 2*dir.Y} // Позиция пешки после рубки
		// Умножая на 2 стандартное перемещение, как раз получаем перемещение после рубки

		// Проверка границ и условий взятия
		if newPos.X < 0 || newPos.X >= 8 || newPos.Y < 0 || newPos.Y >= 8 ||
			visited[enemyPos] || //  Если мы уже рубили данного врага, 2 раз не можем(по правилу)
			field[enemyPos.X][enemyPos.Y].Color_check == player || // Себя рубить нельзя
			field[enemyPos.X][enemyPos.Y].Color_check == "clear" || // Вражеская пешка должна быть
			field[newPos.X][newPos.Y].Color_check != "clear" { // Новая позиция должна быть пустой(за вражеской пешкой)
			continue // Если взятий нет то переходим к следующей итерации
		}
		// Иначе

		// Копируем visited и добавляем новую срубленную шашку
		newVisited := copyVisited(visited)
		newVisited[enemyPos] = true

		// Проверяем превращение(встали на край доски стороны противника -> стали дамкой
		isKing := (player == "white" && newPos.Y == 7) || (player == "black" && newPos.Y == 0)

		// Рекурсивно ищем продолжение
		recursiveBeats := get_checks_beats(newPos, player, field, newVisited) // Проверяем всё также,но уже на след позиции
		if len(recursiveBeats) > 0 {                                          // Случай когда комбинация
			for _, rb := range recursiveBeats {
				rb.Begin_pos = pos
				rb.Will_killed = append([]Cords{enemyPos}, rb.Will_killed...)
				rb.Is_become_king = isKing || rb.Is_become_king
				beats = append(beats, rb)
			}
		} else {
			beats = append(beats, Beat{ // Случай когда обычный ход
				New_pos:        newPos,
				Begin_pos:      pos,
				Will_killed:    []Cords{enemyPos},
				Is_become_king: isKing,
			})
		}
	}
	return beats
}

func get_king_beats(pos Cords, player string, field [8][8]Cell, visited map[Cords]bool) []Beat {
	var beats []Beat
	// Воможные ходы т.к взятия то можем рубить и назад и вперёд не в зависимости от цвета шашки
	directions := []Cords{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}}
	for _, dir := range directions {
		currentVisited := copyVisited(visited)
		killed := Cords{-1, -1}
		hasKilled := false

		for step := 1; ; step++ {
			checkPos := Cords{pos.X + step*dir.X, pos.Y + step*dir.Y}                   // проверяем все возможные клетки хода дамки
			if checkPos.X < 0 || checkPos.X >= 8 || checkPos.Y < 0 || checkPos.Y >= 8 { // выход за пределы
				break
			}

			cell := field[checkPos.X][checkPos.Y]
			if cell.Color_check == "clear" {
				if hasKilled {
					// После удара дамка может встать на любую клетку за побитой
					newVisited := copyVisited(currentVisited)
					newVisited[killed] = true

					recursiveBeats := get_king_beats(checkPos, player, field, newVisited) // Комбинации
					if len(recursiveBeats) > 0 {                                          // Случай когда комбинация
						for _, rb := range recursiveBeats {
							rb.Begin_pos = pos
							rb.Will_killed = append([]Cords{killed}, rb.Will_killed...)
							beats = append(beats, rb)
						}
					} else {
						beats = append(beats, Beat{ // Случай когда обычный ход
							New_pos:        checkPos,
							Begin_pos:      pos,
							Will_killed:    []Cords{killed},
							Is_become_king: true,
						})
					}
				}
				continue
			} else if cell.Color_check == player || currentVisited[checkPos] { // Посещали или на клетке свой
				break
			} else {
				// Встречена вражеская шашка
				if hasKilled { // Живого рубить дважды нельзя т.ч выходим из цикла
					break
				}
				killed = checkPos // Иначе рубим спокойно
				hasKilled = true
			}
		}
	}

	return beats
}

func Get_beats(player string, field [8][8]Cell) []Beat {
	pieces := give_positions_checks(player, field)
	var allBeats []Beat
	initialVisited := make(map[Cords]bool)

	for _, piece := range pieces {
		var beats []Beat
		if piece.Is_king {
			beats = get_king_beats(piece.Pos, player, field, initialVisited)
		} else {
			beats = get_checks_beats(piece.Pos, player, field, initialVisited)
		}
		allBeats = append(allBeats, beats...)
	}

	return allBeats
}

func copyVisited(currentVisited map[Cords]bool) map[Cords]bool { //Созадние копии Мапы убитых шашек(нужна для правильной работы)
	newMap := make(map[Cords]bool)
	for cord, is_visit := range currentVisited {
		newMap[cord] = is_visit
	}
	return newMap
}

func king_pos(x int, y int, field [8][8]Cell) []Cords {
	x1, y1 := x, y
	var pos []Cords
	for x1-1 >= 0 && y1-1 >= 0 {
		x1 = x1 - 1
		y1 = y1 - 1
		if field[x1][y1].Color_check == "clear" {
			pos = append(pos, Cords{x1, y1})
		} else {
			break
		}
	}
	x1, y1 = x, y
	for x1+1 <= 7 && y1+1 <= 7 {
		x1 = x1 + 1
		y1 = y1 + 1
		if field[x1][y1].Color_check == "clear" {
			pos = append(pos, Cords{x1, y1})
		} else {
			break
		}
	}
	x1, y1 = x, y
	for y1-1 >= 0 && x1+1 <= 7 {
		x1 = x1 + 1
		y1 = y1 - 1
		if field[x1][y1].Color_check == "clear" {
			pos = append(pos, Cords{x1, y1})
		} else {
			break
		}
	}
	x1, y1 = x, y
	for x1-1 >= 0 && y1+1 <= 7 {
		x1 = x1 - 1
		y1 = y1 + 1
		if field[x1][y1].Color_check == "clear" {
			pos = append(pos, Cords{x1, y1})
		} else {
			break
		}
	}
	return pos
}
func Is_Can_move(Positions []check_and_kor, field [8][8]Cell, Player string) bool {
	Flag := false
	for _, check := range Positions {
		if Player == "white" {
			x, y := check.Pos.X, check.Pos.Y
			if check.Is_king == false {
				if x-1 >= 0 && y+1 < 8 {
					if field[x-1][y+1].Color_check == "clear" {
						Flag = true
						break
					}
				}
				if x+1 < 8 && y+1 < 8 {
					if field[x+1][y+1].Color_check == "clear" {
						Flag = true
						break
					}
				}
			} else {
				if len(king_pos(x, y, field)) != 0 {
					Flag = true
					break
				}
			}
		}
		if Player == "black" {
			x, y := check.Pos.X, check.Pos.Y
			if check.Is_king == false {
				if x-1 >= 0 && y-1 >= 0 {
					if field[x-1][y-1].Color_check == "clear" {
						Flag = true
						break
					}
				}
				if x+1 < 8 && y-1 >= 0 {
					if field[x+1][y-1].Color_check == "clear" {
						Flag = true
						break
					}
				}
			} else {
				if len(king_pos(x, y, field)) != 0 {
					Flag = true
					break
				}
			}
		}
	}
	return Flag
}

func change_player(Player string) (New_player string) {
	if Player == "black" {
		return "white"
	} else {
		return "black"
	}
}

const (
	cellSize      = 80
	board_Size    = 640
	Screen_size_y = 705
	Screen_size_x = 640
)

func (game *Game) Draw(screen *ebiten.Image) {
	if game.Menu.Active {
		// Отрисовка фона меню
		ebitenutil.DrawRect(screen, 0, 0, float64(board_Size), float64(Screen_size_y), color.RGBA{50, 50, 50, 255})

		// Рисуем медведя
		drawBear(screen)

		// Остальной код отрисовки кнопок и текста
		startBtn := game.Menu.StartButton
		continueBtn := game.Menu.ContinueButton
		exitBtn := game.Menu.ExitButton
		rulesBtn := game.Menu.RulesButton

		ebitenutil.DrawRect(screen, float64(startBtn.Min.X), float64(startBtn.Min.Y),
			float64(startBtn.Dx()), float64(startBtn.Dy()), color.RGBA{100, 200, 100, 255})
		ebitenutil.DrawRect(screen, float64(continueBtn.Min.X), float64(continueBtn.Min.Y),
			float64(startBtn.Dx()), float64(continueBtn.Dy()), color.RGBA{49, 125, 242, 255})
		ebitenutil.DrawRect(screen, float64(rulesBtn.Min.X), float64(rulesBtn.Min.Y),
			float64(rulesBtn.Dx()), float64(rulesBtn.Dy()), color.RGBA{100, 100, 200, 255})
		ebitenutil.DrawRect(screen, float64(exitBtn.Min.X), float64(exitBtn.Min.Y),
			float64(exitBtn.Dx()), float64(exitBtn.Dy()), color.RGBA{200, 100, 100, 255})

		// Текст на кнопках
		text.Draw(screen, "Начать игру", russianFont, startBtn.Min.X+48, startBtn.Min.Y+30, color.White)
		text.Draw(screen, "Продолжить", russianFont, continueBtn.Min.X+50, continueBtn.Min.Y+30, color.White)
		text.Draw(screen, "Правила", russianFont, rulesBtn.Min.X+50, rulesBtn.Min.Y+30, color.White)
		text.Draw(screen, "Выход", russianFont, exitBtn.Min.X+50, exitBtn.Min.Y+30, color.White)

		// Заголовок
		text.Draw(screen, "РУССКИЕ ШАШКИ", russianFont, board_Size/2-55, board_Size/2-70, color.White)

		// Название команды
		text.Draw(screen, "©Ebiten Software (Ebisoft)", russianFont, board_Size/100, Screen_size_y/1-10, color.White)
		return
	}
	if game.RulesScreen.Active || game.ShowRulesInGame {
		// Отрисовка фона
		screen.Fill(color.RGBA{40, 40, 60, 255})

		// Отрисовка декоративных элементов
		ebitenutil.DrawRect(screen, 0, 0, float64(board_Size), 5, color.RGBA{100, 100, 150, 255})
		ebitenutil.DrawRect(screen, 0, float64(board_Size-5), float64(board_Size), 5, color.RGBA{100, 100, 150, 255})

		// Отрисовка заголовка
		title := "Правила игры в русские шашки"
		titleWidth := text.BoundString(game.RulesScreen.TitleFont, title).Dx()
		text.Draw(screen, title, game.RulesScreen.TitleFont,
			(board_Size-titleWidth)/2, 50-int(game.RulesScreen.ScrollY),
			color.RGBA{255, 215, 0, 255}) // Золотой цвет для заголовка

		// Отрисовка текста правил
		for i, line := range game.RulesScreen.TextLines {
			yPos := 75 + i*25 - int(game.RulesScreen.ScrollY)

			// Пропускаем строки, которые выходят за границы экрана
			if yPos < 70 || yPos > board_Size {
				continue
			}

			// Разное оформление для разных типов строк
			if strings.HasPrefix(line, "Правила") || strings.HasPrefix(line, "Основные") {
				// Подзаголовки
				text.Draw(screen, line, game.RulesScreen.TitleFont, 20, yPos, color.RGBA{200, 200, 255, 255})
			} else {
				// Обычный текст
				text.Draw(screen, line, game.RulesScreen.RegularFont, 20, yPos, color.RGBA{255, 255, 255, 255})
			}
		}

		// Кнопка "Назад"
		backBtn := game.RulesScreen.BackButton
		ebitenutil.DrawRect(screen, float64(backBtn.Min.X), float64(backBtn.Min.Y),
			float64(backBtn.Dx()), float64(backBtn.Dy()), color.RGBA{70, 70, 120, 255})
		ebitenutil.DrawRect(screen, float64(backBtn.Min.X)+3, float64(backBtn.Min.Y)+3,
			float64(backBtn.Dx())-6, float64(backBtn.Dy())-6, color.RGBA{100, 100, 150, 255})
		text.Draw(screen, "Назад", game.RulesScreen.RegularFont, backBtn.Min.X+45, backBtn.Min.Y+30, color.White)

		if !game.RulesScreen.Active {
			// Кнопка закрытия (только при показе в игре)
			closeBtn := image.Rect(board_Size/2-100, board_Size-80, board_Size/2+100, board_Size-12)
			ebitenutil.DrawRect(screen, float64(closeBtn.Min.X), float64(closeBtn.Min.Y),
				float64(closeBtn.Dx()), float64(closeBtn.Dy()), color.RGBA{40, 40, 60, 255})
			text.Draw(screen, "Чтобы закрыть нажмите (Tab)", game.RulesScreen.RegularFont,
				closeBtn.Min.X+15, closeBtn.Min.Y+30, color.White)
		}
		return
	}

	// Отрисовка поля
	for y := 0; y < 8; y++ {
		screenY := 7 - y // переворот по вертикали

		for x := 0; x < 8; x++ {
			// Цвет клетки
			var cellColor color.Color
			switch game.Field[x][y].Color_cell {
			case "white":
				cellColor = color.RGBA{240, 217, 181, 255}
			case "black":
				cellColor = color.RGBA{181, 136, 99, 255}
			case "green":
				cellColor = color.RGBA{144, 238, 144, 255}
			case "red":
				cellColor = color.RGBA{255, 99, 71, 255}
			}

			// Отрисовка клетки
			ebitenutil.DrawRect(screen, float64(x*cellSize), float64(screenY*cellSize), cellSize, cellSize, cellColor)

			// Отрисовка шашки
			if game.Field[x][y].Color_check != "clear" {
				var checkerColor color.Color
				if game.Field[x][y].Color_check == "white" {
					checkerColor = color.White
				} else {
					checkerColor = color.Black
				}

				cx := float64(x*cellSize) + cellSize/2
				cy := float64(screenY*cellSize) + cellSize/2
				radius := float64(cellSize) / 2 * 0.8

				ebitenutil.DrawCircle(screen, cx, cy, radius, checkerColor)

				if game.Field[x][y].Is_king {
					var crownColor color.Color
					if game.Field[x][y].Color_check == "white" {
						crownColor = color.Black
					} else {
						crownColor = color.White
					}
					text.Draw(screen, "Д", russianFont, int(cx)-3, int(cy)+5, crownColor)
				}
			}
		}
	}

	// Рамка вокруг выбранной шашки
	if len(game.Clicks) == 1 {
		x, y := game.Clicks[0].X, game.Clicks[0].Y
		screenY := 7 - y
		ebitenutil.DrawRect(screen, float64(x*cellSize), float64(screenY*cellSize), cellSize, 3, color.RGBA{255, 215, 0, 255})
		ebitenutil.DrawRect(screen, float64(x*cellSize), float64((screenY+1)*cellSize)-3, cellSize, 3, color.RGBA{255, 215, 0, 255})
		ebitenutil.DrawRect(screen, float64(x*cellSize), float64(screenY*cellSize), 3, cellSize, color.RGBA{255, 215, 0, 255})
		ebitenutil.DrawRect(screen, float64((x+1)*cellSize)-3, float64(screenY*cellSize), 3, cellSize, color.RGBA{255, 215, 0, 255})
	}

	// Отрисовка кнопок в игре
	ebitenutil.DrawRect(screen, 0, 640, 720, 80, color.RGBA{234, 117, 0, 255}) // Фон для места для кнопок
	ebitenutil.DrawRect(screen, 0, 640, 720, 5, color.RGBA{82, 86, 16, 255})   //Верхняя граница
	ebitenutil.DrawRect(screen, 0, 701, 720, 5, color.RGBA{82, 86, 16, 255})   // Нижняя граница

	ExitBtn := game.Buttons_in_game.Exit_Botton
	RulesBtn := game.Buttons_in_game.Rule_Button
	SaveBtn := game.Buttons_in_game.Save_Button
	TieBtn := game.Buttons_in_game.Tie_Button

	ebitenutil.DrawRect(screen, float64(ExitBtn.Min.X), float64(ExitBtn.Min.Y),
		float64(ExitBtn.Dx()), float64(ExitBtn.Dy()), color.RGBA{138, 154, 91, 255})
	text.Draw(screen, "Exit", game.Buttons_in_game.RegularFont, ExitBtn.Min.X+15, ExitBtn.Min.Y+25, color.White)

	ebitenutil.DrawRect(screen, float64(RulesBtn.Min.X), float64(RulesBtn.Min.Y),
		float64(RulesBtn.Dx()), float64(RulesBtn.Dy()), color.RGBA{138, 154, 91, 255})
	text.Draw(screen, "Rules", game.Buttons_in_game.RegularFont, RulesBtn.Min.X+10, RulesBtn.Min.Y+25, color.White)

	ebitenutil.DrawRect(screen, float64(SaveBtn.Min.X), float64(SaveBtn.Min.Y),
		float64(SaveBtn.Dx()), float64(SaveBtn.Dy()), color.RGBA{138, 154, 91, 255})
	text.Draw(screen, "Save", game.Buttons_in_game.RegularFont, SaveBtn.Min.X+15, SaveBtn.Min.Y+25, color.White)

	ebitenutil.DrawRect(screen, float64(TieBtn.Min.X), float64(TieBtn.Min.Y),
		float64(TieBtn.Dx()), float64(TieBtn.Dy()), color.RGBA{138, 154, 91, 255})
	text.Draw(screen, "Tie(ничья)", game.Buttons_in_game.RegularFont, TieBtn.Min.X+5, TieBtn.Min.Y+25, color.White)

	if game.Proccesed_Exit_Button { // Нажата кнопка выхода из игры
		// Полупрозрачный черный фон
		ebitenutil.DrawRect(screen, 0, 0, float64(board_Size), float64(board_Size), color.RGBA{0, 0, 0, 200})

		Exit_Text := "Вы действительно хотите выйти из игры ?"
		text.Draw(screen, Exit_Text, russianFont, board_Size/2-130, board_Size/2-25, color.White)
		text.Draw(screen, "Y- да N- нет", russianFont, board_Size/2-40, board_Size/2+20, color.White)
		return
	}

	if game.Proccesed_Tie_Button { // Нажата кнопка ничьи
		ebitenutil.DrawRect(screen, 0, 0, float64(board_Size), float64(board_Size), color.RGBA{0, 0, 0, 200})
		Tie_Text := "Оба игрока согласны на ничью ?"
		text.Draw(screen, Tie_Text, russianFont, board_Size/2-105, board_Size/2-25, color.White)
		text.Draw(screen, "Y- да N- нет", russianFont, board_Size/2-40, board_Size/2+20, color.White)
		return
	}

	if game.MassageScreen.Active {
		// Полупрозрачный черный фон
		ebitenutil.DrawRect(screen, 0, 0, float64(board_Size), float64(board_Size), color.RGBA{0, 0, 0, 200})

		// Текст победителя
		Text := ""
		if game.MassageScreen.Winner == "white" {
			Text = "БЕЛЫЕ ПОБЕДИЛИ!"
		} else if game.MassageScreen.Winner == "black" {
			Text = "ЧЁРНЫЕ ПОБЕДИЛИ!"
		} else {
			Text = "             НИЧЬЯ :|"
		}
		text.Draw(screen, Text, russianFont, board_Size/2-60, board_Size/2-20, color.White)
		text.Draw(screen, "Нажмите ESC для возврата в меню", russianFont, board_Size/2-100, board_Size/2+20, color.White)
	}

}

func (game *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return Screen_size_x, Screen_size_y
}
func initGame() *Game {
	game := &Game{
		Width:           board_Size,
		Height:          board_Size,
		Now_player:      "white",
		ClickDelay:      200 * 1000000,
		Winner:          "None",
		InMenu:          true,
		ShowRulesInGame: false,
		Menu: Menu{
			StartButton:    image.Rect(board_Size/2-100, board_Size/2-50, board_Size/2+100, board_Size/2),
			ContinueButton: image.Rect(board_Size/2-100, board_Size/2+20, board_Size/2+100, board_Size/2+70),
			RulesButton:    image.Rect(board_Size/2-100, board_Size/2+90, board_Size/2+100, board_Size/2+140),
			ExitButton:     image.Rect(board_Size/2-100, board_Size/2+160, board_Size/2+100, board_Size/2+210),
			Active:         true,
		},
		RulesScreen: RulesScreen{
			BackButton:  image.Rect(board_Size/2-100, board_Size-80, board_Size/2+100, board_Size-30),
			TextLines:   strings.Split(rulesText, "\n"),
			TitleFont:   russianFont2,
			RegularFont: russianFont,
		},
		Buttons_in_game: Game_Buttons{
			Save_Button: image.Rect(10, 697, 70, 650),
			Exit_Botton: image.Rect(80, 697, 140, 650),
			Rule_Button: image.Rect(150, 697, 210, 650),
			Tie_Button:  image.Rect(220, 697, 295, 650),
			RegularFont: russianFont,
		},
	}
	{
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				// Устанавливаем цвет клетки (шахматная доска)
				if (x+y)%2 == 0 {
					game.Field[x][y].Color_cell = "black"
					game.Field[x][y].Color_check = "clear"
				} else {
					game.Field[x][y].Color_cell = "white"
					game.Field[x][y].Color_check = "clear"
				}

				// Расставляем шашки
				if y < 3 && (x+y)%2 == 0 {
					game.Field[x][y].Color_check = "white"
					game.Field[x][y].Is_king = false
				} else if y > 4 && (x+y)%2 == 0 {
					game.Field[x][y].Color_check = "black"
					game.Field[x][y].Is_king = false
				} else {
					game.Field[x][y].Color_check = "clear"
					game.Field[x][y].Is_king = false
				}
			}
		}
	}
	return game
}
func init() {
	fontBytes, err := ioutil.ReadFile("fonts/Open_Sans/static/OpenSans_Condensed-Bold.ttf")
	if err != nil {
		log.Fatal(err)
	}
	f, err := opentype.Parse(fontBytes)
	if err != nil {
		log.Fatal(err)
	}
	russianFont, err = opentype.NewFace(f, &opentype.FaceOptions{
		Size: 15,
		DPI:  72,
	})
	if err != nil {
		log.Fatal(err)
	}
	russianFont2, err = opentype.NewFace(f, &opentype.FaceOptions{
		Size: 24,
		DPI:  72,
	})
	if err != nil {
		log.Fatal(err)
	}
}
func drawBear(screen *ebiten.Image) {
	// Тело медведя — большой круг
	ebitenutil.DrawCircle(screen, float64(board_Size/2), float64(board_Size/1.6), 210, color.RGBA{139, 69, 19, 255})

	// Голова — меньший круг сверху
	ebitenutil.DrawCircle(screen, float64(board_Size/2), float64(board_Size/1-400), 100, color.RGBA{139, 69, 19, 255})

	// Уши — два маленьких круга
	ebitenutil.DrawCircle(screen, float64(board_Size/2-44), float64(board_Size/2-180), 30, color.RGBA{139, 69, 19, 255})
	ebitenutil.DrawCircle(screen, float64(board_Size/2+44), float64(board_Size/2-180), 30, color.RGBA{139, 69, 19, 255})

	// Глаза — два маленьких белых кружка
	ebitenutil.DrawCircle(screen, float64(board_Size/2-15), float64(board_Size/2-115), 10, color.White)
	ebitenutil.DrawCircle(screen, float64(board_Size/2+15), float64(board_Size/2-115), 10, color.White)

	// Нос — маленький черный кружок
	ebitenutil.DrawCircle(screen, float64(board_Size/2), float64(board_Size/2-100), 8, color.Black)

}
func main() {
	game := initGame()
	ebiten.SetTPS(17)

	ebiten.SetWindowSize(Screen_size_x, Screen_size_y)
	ebiten.SetWindowTitle("Русские шашки")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
