package task

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Board struct {
	help     help.Model
	loaded   bool
	focused  status
	cols     []column
	quitting bool
	service  *Service
}

func NewBoard() (*Board, error) {
	help := help.New()
	help.ShowAll = true

	service, err := NewService()
	if err != nil {
		return nil, err
	}

	board := &Board{help: help, focused: todo, service: service}
	board.initLists()
	return board, nil
}

func (m *Board) Init() tea.Cmd {
	return nil
}

func (m *Board) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		var cmd tea.Cmd
		var cmds []tea.Cmd
		m.help.Width = msg.Width - Margin
		for i := 0; i < len(m.cols); i++ {
			var res tea.Model
			res, cmd = m.cols[i].Update(msg)
			m.cols[i] = res.(column)
			cmds = append(cmds, cmd)
		}
		m.loaded = true
		return m, tea.Batch(cmds...)
	case Form:
		task := msg.CreateTask()
		if msg.index == AppendIndex {
			// Creating new task
			m.service.CreateTask(task)
		} else {
			// Editing existing task - get the original task ID
			originalTask := m.cols[m.focused].list.Items()[msg.index].(Task)
			task = NewTaskWithID(originalTask.ID(), task.Status(), task.Title(), task.Description())
			m.service.UpdateTask(task)
		}
		return m, m.cols[m.focused].Set(msg.index, task)
	case moveMsg:
		m.service.UpdateTask(msg.Task)
		return m, m.cols[m.focused.getNext()].Set(AppendIndex, msg.Task)
	case deleteMsg:
		m.service.DeleteTask(msg.Task.ID())
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			if m.service != nil {
				m.service.Close()
			}
			return m, tea.Quit
		case key.Matches(msg, keys.Left):
			m.cols[m.focused].Blur()
			m.focused = m.focused.getPrev()
			m.cols[m.focused].Focus()
		case key.Matches(msg, keys.Right):
			m.cols[m.focused].Blur()
			m.focused = m.focused.getNext()
			m.cols[m.focused].Focus()
		}
	}
	res, cmd := m.cols[m.focused].UpdateWithBoard(msg, m)
	if _, ok := res.(column); ok {
		m.cols[m.focused] = res.(column)
	} else {
		return res, cmd
	}
	return m, cmd
}

func (m *Board) View() string {
	if m.quitting {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Goodbye! 👋")
	}
	if !m.loaded {
		loadingStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginTop(1).
			MarginLeft(2)
		return loadingStyle.Render("🔄 Loading your tasks...")
	}

	// Add spacing between columns
	columnGap := lipgloss.NewStyle().Width(2).Render("")

	board := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.cols[todo].View(),
		columnGap,
		m.cols[inProgress].View(),
		columnGap,
		m.cols[done].View(),
	)

	// Style the board with margin to align with help box
	boardStyle := lipgloss.NewStyle().
		Margin(1, 2)

	styledBoard := boardStyle.Render(board)

	// Style the help section
	helpStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		MarginLeft(2)

	styledHelp := helpStyle.Render(m.help.View(keys))

	return lipgloss.JoinVertical(lipgloss.Left, styledBoard, styledHelp)
}

func (b *Board) initLists() {
	b.cols = []column{
		newColumn(todo),
		newColumn(inProgress),
		newColumn(done),
	}
	b.cols[todo].list.Title = ColumnTitleTodo
	b.cols[inProgress].list.Title = ColumnTitleInProgress
	b.cols[done].list.Title = ColumnTitleDone

	b.loadTasksFromDB()
}

func (b *Board) loadTasksFromDB() {
	if b.service == nil {
		b.loadDefaultTasks()
		return
	}

	tasks, err := b.service.LoadAllTasks()
	if err != nil {
		b.loadDefaultTasks()
		return
	}

	var todoItems, inProgressItems, doneItems []list.Item

	for _, task := range tasks {
		switch task.Status() {
		case todo:
			todoItems = append(todoItems, task)
		case inProgress:
			inProgressItems = append(inProgressItems, task)
		case done:
			doneItems = append(doneItems, task)
		}
	}

	b.cols[todo].list.SetItems(todoItems)
	b.cols[inProgress].list.SetItems(inProgressItems)
	b.cols[done].list.SetItems(doneItems)
}

func (b *Board) loadDefaultTasks() {
	b.cols[todo].list.SetItems([]list.Item{
		NewTask(todo, "buy milk", "strawberry milk"),
		NewTask(todo, "eat sushi", "negitoro roll, miso soup, rice"),
		NewTask(todo, "fold laundry", "or wear wrinkly t-shirts"),
	})
	b.cols[inProgress].list.SetItems([]list.Item{
		NewTask(inProgress, "write code", "don't worry, it's Go"),
	})
	b.cols[done].list.SetItems([]list.Item{
		NewTask(done, "stay cool", "as a cucumber"),
	})
}
