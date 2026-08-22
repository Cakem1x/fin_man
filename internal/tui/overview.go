package tui

import (
	"fmt"
	"sort"
	"time"

	"github.com/Cakem1x/fin_man/internal/model"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1).
			MarginBottom(1)
	filterStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")).MarginBottom(1)
)

type timeFilter int

const (
	filterCurrentMonth timeFilter = iota
	filterCurrentYear
	filterAllTime
)

func (f timeFilter) String() string {
	switch f {
	case filterCurrentMonth:
		return "Current Month"
	case filterCurrentYear:
		return "Current Year"
	case filterAllTime:
		return "All Time"
	default:
		return "Unknown"
	}
}

type categoryItem struct {
	name       string
	totalCents int64
}

func (i categoryItem) Title() string       { return i.name }
func (i categoryItem) Description() string { return fmt.Sprintf("%.2f", float64(i.totalCents)/100.0) }
func (i categoryItem) FilterValue() string { return i.name }

type OverviewModel struct {
	transactions []model.Transaction
	filter       timeFilter

	catList   list.Model
	txTable   table.Model

	activePane int // 0 for list, 1 for table

	width  int
	height int
}

func NewOverviewModel(transactions []model.Transaction) OverviewModel {
	// Setup list
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Categories"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	// Setup table
	columns := []table.Column{
		{Title: "Date", Width: 12},
		{Title: "Payee", Width: 25},
		{Title: "Amount", Width: 10},
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(false),
		table.WithHeight(10),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := OverviewModel{
		transactions: transactions,
		filter:       filterCurrentMonth,
		catList:      l,
		txTable:      t,
		activePane:   0, // focus list first
	}
	m.updateData()
	return m
}

func (m *OverviewModel) Init() tea.Cmd {
	return nil
}

func (m *OverviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		h, v := appStyle.GetFrameSize()
		contentWidth := m.width - h
		contentHeight := m.height - v - 4 // subtract title/filter height

		// 1/3 for categories, 2/3 for transactions
		listWidth := contentWidth / 3
		tableWidth := contentWidth - listWidth - 2 // some gap

		m.catList.SetSize(listWidth, contentHeight)

		// Resize table columns proportionally
		m.txTable.SetHeight(contentHeight - 2)
		cols := m.txTable.Columns()
		if len(cols) > 0 {
			cols[0].Width = 10
			cols[2].Width = 10
			cols[1].Width = tableWidth - 20 - 4 // minus borders/padding
			m.txTable.SetColumns(cols)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			// Toggle filter
			m.filter = (m.filter + 1) % 3
			m.updateData()
			return m, nil
		case "h", "left":
			m.activePane = 0
			m.catList.Title = "Categories (Focused)"
			m.txTable.Focus() // unfocus hack since bool arg isn't there in some versions
			m.txTable.Blur()
		case "l", "right":
			m.activePane = 1
			m.catList.Title = "Categories"
			m.txTable.Focus()
		}
	}

	// Route events to the active pane
	previousSelectedCat := m.catList.Index()

	if m.activePane == 0 {
		m.catList, cmd = m.catList.Update(msg)
		cmds = append(cmds, cmd)

		// If category changed, update table
		if m.catList.Index() != previousSelectedCat {
			m.updateTableData()
		}
	} else {
		m.txTable, cmd = m.txTable.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *OverviewModel) updateData() {
	now := time.Now()
	var filtered []model.Transaction

	for _, tx := range m.transactions {
		keep := false
		switch m.filter {
		case filterAllTime:
			keep = true
		case filterCurrentYear:
			if tx.Date.Year() == now.Year() {
				keep = true
			}
		case filterCurrentMonth:
			if tx.Date.Year() == now.Year() && tx.Date.Month() == now.Month() {
				keep = true
			}
		}
		if keep {
			filtered = append(filtered, tx)
		}
	}

	// Group by category
	catTotals := make(map[string]int64)
	for _, tx := range filtered {
		catName := "Uncategorized"
		if tx.CategoryName != nil && *tx.CategoryName != "" {
			catName = *tx.CategoryName
		}
		catTotals[catName] += tx.AmountCents
	}

	var items []list.Item
	for name, total := range catTotals {
		items = append(items, categoryItem{name: name, totalCents: total})
	}

	// Sort items by total amount (descending)
	sort.Slice(items, func(i, j int) bool {
		return items[i].(categoryItem).totalCents > items[j].(categoryItem).totalCents
	})

	m.catList.SetItems(items)
	m.updateTableData()
}

func (m *OverviewModel) updateTableData() {
	selectedItem := m.catList.SelectedItem()
	if selectedItem == nil {
		m.txTable.SetRows([]table.Row{})
		return
	}
	cat := selectedItem.(categoryItem)

	now := time.Now()
	var rows []table.Row
	for _, tx := range m.transactions {
		// Apply same time filter
		keep := false
		switch m.filter {
		case filterAllTime:
			keep = true
		case filterCurrentYear:
			if tx.Date.Year() == now.Year() {
				keep = true
			}
		case filterCurrentMonth:
			if tx.Date.Year() == now.Year() && tx.Date.Month() == now.Month() {
				keep = true
			}
		}

		if !keep {
			continue
		}

		txCat := "Uncategorized"
		if tx.CategoryName != nil && *tx.CategoryName != "" {
			txCat = *tx.CategoryName
		}

		if txCat == cat.name {
			amountStr := fmt.Sprintf("%.2f %s", float64(tx.AmountCents)/100.0, tx.Currency)
			rows = append(rows, table.Row{
				tx.Date.Format("2006-01-02"),
				tx.Payee,
				amountStr,
			})
		}
	}

	// Sort rows by date descending
	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] > rows[j][0]
	})

	m.txTable.SetRows(rows)
}

func (m *OverviewModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	title := titleStyle.Render(" Finance Overview ")
	filterText := filterStyle.Render(fmt.Sprintf("Filter: %s (press tab to toggle)", m.filter.String()))

	header := lipgloss.JoinVertical(lipgloss.Left, title, filterText)

	listView := lipgloss.NewStyle().MarginRight(2).Render(m.catList.View())
	tableView := m.txTable.View()

	panes := lipgloss.JoinHorizontal(lipgloss.Top, listView, tableView)

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("\n[q] Quit • [h/l] Switch Panes • [j/k] Navigate")

	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, panes, help))
}
