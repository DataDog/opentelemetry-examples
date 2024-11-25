package gameoflife

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	gameoflifepb "github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/pb"

	"go.uber.org/zap/zaptest"
)

func TestRun(t *testing.T) {
	var tests = []struct {
		board         string
		numGens       int32
		responseCode  gameoflifepb.ResponseCode
		responseBoard string
	}{
		{"[[1]]", 1, gameoflifepb.ResponseCode_OK, "[[0]]"},
		{"[[1]]", 100, gameoflifepb.ResponseCode_OK, "[[0]]"},
		{"[[1,1],[1,0]]", 1, gameoflifepb.ResponseCode_OK, "[[1,1],[1,1]]"},
		{"[[1,1],[1,0]]", 10, gameoflifepb.ResponseCode_OK, "[[1,1],[1,1]]"},
		{"[[0,1,0],[0,0,1],[1,1,1],[0,0,0]]", 1, gameoflifepb.ResponseCode_OK, "[[0,0,0],[1,0,1],[0,1,1],[0,1,0]]"},
		{"[[0,1,0],[0,0,1],[1,1,1],[0,0,0]]", 2, gameoflifepb.ResponseCode_OK, "[[0,0,0],[0,0,1],[1,0,1],[0,1,1]]"},
		{"[[0,1,0],[0,0,1],[1,1,1],[0,0,0]]", 3, gameoflifepb.ResponseCode_OK, "[[0,0,0],[0,1,0],[0,0,1],[0,1,1]]"},
		{"[[0,1,0],[0,0,1],[1,1,1],[0,0,0]]", 4, gameoflifepb.ResponseCode_OK, "[[0,0,0],[0,0,0],[0,0,1],[0,1,1]]"},
		{"[[0,1,0],[0,0,1],[1,1,1],[0,0,0]]", 20, gameoflifepb.ResponseCode_OK, "[[0,0,0],[0,0,0],[0,1,1],[0,1,1]]"},
		{"[[0,1,0],[0,1,0],[0,1,0]]", 1, gameoflifepb.ResponseCode_OK, "[[0,0,0],[1,1,1],[0,0,0]]"},
		{"[[0,1,0],[0,1,0],[0,1,0]]", 2, gameoflifepb.ResponseCode_OK, "[[0,1,0],[0,1,0],[0,1,0]]"},
	}
	for _, tt := range tests {
		testname := fmt.Sprintf("%+v", &tt)
		t.Run(testname, func(t *testing.T) {
			ans, err := Run(context.Background(), &gameoflifepb.GameRequest{
				Board:   tt.board,
				NumGens: tt.numGens,
			}, zaptest.NewLogger(t))
			if err != nil {
				t.Errorf("Error: %v", err)
			} else if ans.GetBoard() != tt.responseBoard {
				t.Errorf("Got %v, expected %v", ans.Board, tt.responseBoard)
			}
		})
	}

	var errorTests = []struct {
		board        string
		numGens      int32
		responseCode gameoflifepb.ResponseCode
	}{
		{"invalid board", 1, gameoflifepb.ResponseCode_BAD_REQUEST},
		{"[1,1]", 1, gameoflifepb.ResponseCode_BAD_REQUEST},
		{"[[]]", 1, gameoflifepb.ResponseCode_BAD_REQUEST},
		{"[]", 1, gameoflifepb.ResponseCode_BAD_REQUEST},
		{"[1]", 1, gameoflifepb.ResponseCode_BAD_REQUEST},
		{"[[1,0],[1,2]]", 1, gameoflifepb.ResponseCode_BAD_REQUEST},
	}
	for _, tt := range errorTests {
		testname := fmt.Sprintf("%+v", &tt)
		t.Run(testname, func(t *testing.T) {
			ans, err := Run(context.Background(), &gameoflifepb.GameRequest{
				Board:   tt.board,
				NumGens: tt.numGens,
			}, zaptest.NewLogger(t))
			if err == nil {
				t.Errorf("Error not found: %v", err)
			} else if ans.Code != tt.responseCode {
				t.Errorf("Got %v, expected %v", ans.Code, tt.responseCode)
			}
		})
	}
}

func TestCalcLiveNeighbors(t *testing.T) {
	var tests = []struct {
		row            int
		col            int
		board          [][]int
		expectedResult int
	}{
		{0, 0, [][]int{{1}}, 0},
		{0, 0, [][]int{{1, 1}, {1, 0}}, 2},
		{0, 1, [][]int{{1, 1}, {1, 0}}, 2},
		{1, 0, [][]int{{1, 1}, {1, 0}}, 2},
		{1, 1, [][]int{{1, 1}, {1, 0}}, 3},
		{1, 1, [][]int{{0, 1, 0}, {0, 0, 1}, {1, 1, 1}, {0, 0, 0}}, 5},
		{3, 2, [][]int{{0, 1, 0}, {0, 0, 1}, {1, 1, 1}, {0, 0, 0}}, 2},
	}
	for _, tt := range tests {
		testname := fmt.Sprintf("%v,%v,%v,%v", &tt.row, &tt.col, &tt.board, &tt.expectedResult)
		t.Run(testname, func(t *testing.T) {
			ans := calcLiveNeighbors(tt.row, tt.col, tt.board)
			if ans != tt.expectedResult {
				t.Errorf("Got %v, expected %v", ans, tt.expectedResult)
			}
		})
	}
}

func TestExecuteRules(t *testing.T) {
	var tests = []struct {
		fromBoard      [][]int
		expectedResult [][]int
	}{
		{[][]int{{1}}, [][]int{{0}}},
		{[][]int{{0}}, [][]int{{0}}},
		{[][]int{{1, 1}, {1, 0}}, [][]int{{1, 1}, {1, 1}}},
		{[][]int{{1, 1}, {1, 1}}, [][]int{{1, 1}, {1, 1}}},
		{[][]int{{0, 1, 0}, {0, 0, 1}, {1, 1, 1}, {0, 0, 0}}, [][]int{{0, 0, 0}, {1, 0, 1}, {0, 1, 1}, {0, 1, 0}}},
		{[][]int{{0, 0, 0}, {1, 0, 1}, {0, 1, 1}, {0, 1, 0}}, [][]int{{0, 0, 0}, {0, 0, 1}, {1, 0, 1}, {0, 1, 1}}},
		{[][]int{{0, 1, 0}, {0, 1, 0}, {0, 1, 0}}, [][]int{{0, 0, 0}, {1, 1, 1}, {0, 0, 0}}},
		{[][]int{{0, 0, 0}, {1, 1, 1}, {0, 0, 0}}, [][]int{{0, 1, 0}, {0, 1, 0}, {0, 1, 0}}},
	}
	for _, tt := range tests {
		testname := fmt.Sprintf("%v,%v", &tt.fromBoard, &tt.expectedResult)
		t.Run(testname, func(t *testing.T) {
			ans := executeRules(tt.fromBoard)
			if !reflect.DeepEqual(tt.expectedResult, ans) {
				t.Errorf("Got %v, expected %v", ans, tt.expectedResult)
			}
		})
	}
}
