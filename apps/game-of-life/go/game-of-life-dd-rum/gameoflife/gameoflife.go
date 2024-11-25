package gameoflife

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	gameoflifepb "github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/pb"

	"go.uber.org/zap"
)

// calcLiveNeighbors Calculates number of live neighbors for the cell at (row, col) on the given board
func calcLiveNeighbors(row int, col int, board [][]int) int {
	sum := 0
	for i := row - 1; i <= row+1; i++ {
		for j := col - 1; j <= col+1; j++ {
			if i >= 0 && i < len(board) && j >= 0 && j < len(board[0]) && !(i == row && j == col) {
				sum += board[i][j]
			}
		}
	}
	return sum
}

// executeRules Returns board resulting from executing rules on the given board
func executeRules(fromBoard [][]int) [][]int {
	toBoard := copyBoard(fromBoard)
	for i := 0; i < len(fromBoard); i++ {
		for j := 0; j < len(fromBoard[0]); j++ {
			liveNeighbors := calcLiveNeighbors(i, j, fromBoard)
			fromCell := fromBoard[i][j]
			if fromCell == 1 {
				if liveNeighbors < 2 {
					toBoard[i][j] = 0
				} else if liveNeighbors <= 3 {
					toBoard[i][j] = 1
				} else {
					toBoard[i][j] = 0
				}
			} else if fromCell == 0 && liveNeighbors == 3 {
				toBoard[i][j] = 1
			}
		}
	}
	return toBoard
}

// copyBoard Returns deep copy of the given board
func copyBoard(board [][]int) [][]int {
	result := make([][]int, len(board))
	for i := range board {
		result[i] = make([]int, len(board[i]))
		copy(result[i], board[i])
	}
	return result
}

// validateBoard Returns an error if the given board is not valid
func validateBoard(board [][]int) error {
	if len(board) < 1 || len(board[0]) < 1 {
		return errors.New("board size must be at least 1x1")
	}
	for _, row := range board {
		if len(row) != len(board[0]) {
			return errors.New("board rows must be equal length")
		}
		for _, cell := range row {
			if cell != 0 && cell != 1 {
				return errors.New("cells can only be 0's or 1's")
			}
		}
	}
	return nil
}

// parseBoard Parses board from given string and return 2D int slice
func parseBoard(data string, logger *zap.Logger) ([][]int, error) {
	board := make([][]int, 1)
	if err := json.Unmarshal([]byte(data), &board); err != nil {
		logger.Error("failed to parse", zap.Error(err))
		return nil, err
	}
	return board, nil
}

func Run(ctx context.Context, gameRequest *gameoflifepb.GameRequest, logger *zap.Logger) (*gameoflifepb.GameResponse, error) {
	fromBoard, err := parseBoard(gameRequest.Board, logger)
	if err != nil {
		logger.Error("Failed to parse board",
			zap.String("board", gameRequest.Board),
			zap.Error(err),
		)
		return &gameoflifepb.GameResponse{
			Code:         gameoflifepb.ResponseCode_BAD_REQUEST,
			ErrorMessage: fmt.Sprintf("Failed to parse: %v", gameRequest.Board),
		}, err
	}
	err = validateBoard(fromBoard)
	if err != nil {
		logger.Error("Invalid board",
			zap.Any("board", fromBoard),
			zap.Error(err),
		)
		return &gameoflifepb.GameResponse{
			Code:         gameoflifepb.ResponseCode_BAD_REQUEST,
			ErrorMessage: fmt.Sprintf("Invalid board: %v", gameRequest.Board),
		}, err
	}
	var toBoard [][]int

	logger.Info("Current board",
		zap.Int("generation", 0),
		zap.Any("board", fromBoard),
	)
	for i := 1; i <= int(gameRequest.NumGens); i++ {
		toBoard = executeRules(fromBoard)
		fromBoard = copyBoard(toBoard)
		logger.Info("Current board",
			zap.Int("generation", i),
			zap.Any("board", toBoard),
		)
	}

	return &gameoflifepb.GameResponse{
		Code:  gameoflifepb.ResponseCode_OK,
		Board: strings.Join(strings.Fields(fmt.Sprint(toBoard)), ","),
	}, nil
}
