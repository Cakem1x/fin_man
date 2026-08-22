package tui

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/Cakem1x/fin_man/internal/categorize"
	"github.com/Cakem1x/fin_man/internal/db"
	"github.com/Cakem1x/fin_man/internal/model"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle    = lipgloss.NewStyle().Padding(1, 2)
	titleStyle  = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1).
			MarginBottom(1)
	filterStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")).MarginBottom(1)

	focusedPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	blurredPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
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
	displayName  string
	fullPath     string
	totalCents   int64
	count        int
	isUnreviewed bool
	isSeparator  bool
}

func (i categoryItem) Title() string {
	if i.isSeparator {
		return "──────────────────"
	}
	return i.displayName
}
func (i categoryItem) Description() string {
	if i.isSeparator {
		return ""
	}
	if i.isUnreviewed {
		return fmt.Sprintf("%d transactions", i.count)
	}
	return fmt.Sprintf("%.2f", float64(i.totalCents)/100.0)
}
func (i categoryItem) FilterValue() string {
	if i.isSeparator {
		return ""
	}
	return i.fullPath
}

type catNode struct {
	name       string
	fullPath   string
	selfCents  int64
	totalCents int64
	children   map[string]*catNode
}

type OverviewModel struct {
	dbConn       *db.DB
	transactions []model.Transaction
	filter       timeFilter

	// Overview pane
	catList    list.Model
	txTable    table.Model
	activePane int // 0 for list, 1 for table

	// Review form overlay
	reviewingTx bool
	reviewingID string
	categories  []string
	tags        []string
	estimator   *categorize.Estimator
	reviewForm  *huh.Form
	reviewState *ReviewFormState

	width  int
	height int
}

func NewOverviewModel(transactions []model.Transaction, dbConn *db.DB) OverviewModel {
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
		{Title: "Category", Width: 15},
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

	ctx := context.Background()
	cats, _ := dbConn.GetAllCategories(ctx)
	catNames := make([]string, len(cats))
	for i, c := range cats {
		catNames[i] = c.Name
	}

	tags, _ := dbConn.GetAllTags(ctx)
	tagNames := make([]string, len(tags))
	for i, tg := range tags {
		tagNames[i] = tg.Name
	}

	categorizedTxs, _ := dbConn.GetCategorizedTransactions(ctx)
	estimator := categorize.NewEstimator(categorizedTxs)

	m := OverviewModel{
		dbConn:       dbConn,
		transactions: transactions,
		filter:       filterCurrentMonth,
		catList:      l,
		txTable:      t,
		activePane:   0,
		categories:   catNames,
		tags:         tagNames,
		estimator:    estimator,
	}
	m.updateData()
	return m
}

func (m *OverviewModel) Init() tea.Cmd {
	return nil
}

func (m *OverviewModel) openReviewForm(tx model.Transaction) {
	suggestion := m.estimator.Estimate(tx)
	state := &ReviewFormState{}
	m.reviewState = state
	m.reviewForm = BuildReviewForm(tx, m.categories, m.tags, suggestion, state)
	m.reviewingID = tx.ID
	m.reviewingTx = true
	m.reviewForm.Init()
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

		listWidth := contentWidth / 3
		tableWidth := contentWidth - listWidth

		borderH, borderV := focusedPaneStyle.GetFrameSize()

		m.catList.SetSize(listWidth-borderH, contentHeight-borderV)

		m.txTable.SetHeight(contentHeight - borderV - 2) // table.Model needs extra -2 for its own header/footer
		cols := m.txTable.Columns()
		if len(cols) > 0 {
			cols[0].Width = 10
			cols[2].Width = 10
			cols[3].Width = 15
			cols[1].Width = tableWidth - borderH - 35 - 4
			m.txTable.SetColumns(cols)
		}

	case tea.KeyMsg:
		if m.reviewingTx {
			// If in review form, we don't process global hotkeys like 'q' or 'tab'
			break
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.filter = (m.filter + 1) % 3
			m.updateData()
			return m, nil
		}
	}

	if !m.reviewingTx {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "h" || msg.String() == "left" {
				m.activePane = 0
				m.catList.Title = "Categories"
				m.txTable.Blur()
			} else if msg.String() == "l" || msg.String() == "right" {
				m.activePane = 1
				m.catList.Title = "Categories"
				m.txTable.Focus()
			}
		}

		previousSelectedCat := m.catList.Index()

		if m.activePane == 0 {
			m.catList, cmd = m.catList.Update(msg)
			cmds = append(cmds, cmd)

			// Skip separator
			if item, ok := m.catList.SelectedItem().(categoryItem); ok && item.isSeparator {
				if msg, ok := msg.(tea.KeyMsg); ok {
					if msg.String() == "j" || msg.String() == "down" {
						m.catList.CursorDown()
					} else if msg.String() == "k" || msg.String() == "up" {
						m.catList.CursorUp()
					}
				}
			}

			if m.catList.Index() != previousSelectedCat {
				m.updateTableData()
			}
		} else {
			// Actually handle enter in table
			if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
				// Let's get the currently displayed txs
				displayed := m.getDisplayedTransactions()
				idx := m.txTable.Cursor()
				if idx >= 0 && idx < len(displayed) {
					m.openReviewForm(displayed[idx])
				}
				return m, nil
			}

			m.txTable, cmd = m.txTable.Update(msg)
			cmds = append(cmds, cmd)
		}
	} else {
		if m.reviewForm != nil {
			var newForm tea.Model
			newForm, cmd = m.reviewForm.Update(msg)
			if f, ok := newForm.(*huh.Form); ok {
				m.reviewForm = f
			}
			cmds = append(cmds, cmd)

			if m.reviewForm.State == huh.StateCompleted {
				res := ExtractReviewResult(m.reviewState)

				if res.Action == ActionSave || res.Action == ActionSaveReviewed {
					ctx := context.Background()
					err := m.dbConn.EnrichTransaction(ctx, m.reviewingID, res.Category, res.Tags, res.Memo, res.Action == ActionSaveReviewed)
					if err != nil {
						log.Printf("failed to save transaction %s: %v", m.reviewingID, err)
					} else {
						if res.Category != "" {
							found := false
							for _, c := range m.categories {
								if c == res.Category {
									found = true
									break
								}
							}
							if !found {
								m.categories = append(m.categories, res.Category)
							}
						}
						for _, t := range res.Tags {
							if t == "" {
								continue
							}
							found := false
							for _, existing := range m.tags {
								if existing == t {
									found = true
									break
								}
							}
							if !found {
								m.tags = append(m.tags, t)
							}
						}
						for i, t := range m.transactions {
							if t.ID == m.reviewingID {
								catName := res.Category
								m.transactions[i].CategoryName = &catName
								m.transactions[i].IsReviewed = (res.Action == ActionSaveReviewed)
								m.transactions[i].Memo = res.Memo
								break
							}
						}
					}
				}
				m.reviewingTx = false
				m.updateData()
			} else if m.reviewForm.State == huh.StateAborted {
				m.reviewingTx = false
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *OverviewModel) updateData() {
	now := time.Now()
	var filteredReviewed []model.Transaction
	var uncatCount int

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

		if !keep {
			continue
		}

		if !tx.IsReviewed {
			uncatCount++
		} else {
			filteredReviewed = append(filteredReviewed, tx)
		}
	}

	rootNodes := make(map[string]*catNode)

	getOrAddNode := func(fullPath string) *catNode {
		parts := strings.Split(fullPath, "/")
		currentLevel := rootNodes
		var currentNode *catNode
		var currentPath string

		for i, part := range parts {
			if i == 0 {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}

			node, exists := currentLevel[part]
			if !exists {
				node = &catNode{
					name:     part,
					fullPath: currentPath,
					children: make(map[string]*catNode),
				}
				currentLevel[part] = node
			}
			currentNode = node
			currentLevel = node.children
		}
		return currentNode
	}

	for _, c := range m.categories {
		getOrAddNode(c)
	}

	for _, tx := range filteredReviewed {
		if tx.CategoryName != nil && *tx.CategoryName != "" {
			node := getOrAddNode(*tx.CategoryName)
			node.selfCents += tx.AmountCents
		}
	}

	var computeTotals func(node *catNode) int64
	computeTotals = func(node *catNode) int64 {
		total := node.selfCents
		for _, child := range node.children {
			total += computeTotals(child)
		}
		node.totalCents = total
		return total
	}

	for _, root := range rootNodes {
		computeTotals(root)
	}

	var flattenTree func(nodes map[string]*catNode, prefix string, isRoot bool) []categoryItem
	flattenTree = func(nodes map[string]*catNode, prefix string, isRoot bool) []categoryItem {
		var sortedNodes []*catNode
		for _, node := range nodes {
			sortedNodes = append(sortedNodes, node)
		}
		sort.Slice(sortedNodes, func(i, j int) bool {
			if sortedNodes[i].totalCents == sortedNodes[j].totalCents {
				return sortedNodes[i].name < sortedNodes[j].name
			}
			return sortedNodes[i].totalCents > sortedNodes[j].totalCents
		})

		var result []categoryItem
		for i, node := range sortedNodes {
			last := i == len(sortedNodes)-1

			nodePrefix := prefix
			childPrefix := prefix

			if !isRoot {
				if last {
					nodePrefix += "└── "
					childPrefix += "    "
				} else {
					nodePrefix += "├── "
					childPrefix += "│   "
				}
			}

			result = append(result, categoryItem{
				displayName: nodePrefix + node.name,
				fullPath:    node.fullPath,
				totalCents:  node.totalCents,
			})

			if len(node.children) > 0 {
				result = append(result, flattenTree(node.children, childPrefix, false)...)
			}
		}
		return result
	}

	var items []list.Item

	// Add Unreviewed at the top
	items = append(items, categoryItem{
		displayName:  "⭐ Unreviewed",
		fullPath:     "__unreviewed__",
		count:        uncatCount,
		isUnreviewed: true,
	})

	// Add separator
	items = append(items, categoryItem{
		isSeparator: true,
	})

	flatItems := flattenTree(rootNodes, "", true)
	for _, fi := range flatItems {
		items = append(items, fi)
	}

	m.catList.SetItems(items)
	m.updateTableData()
}

func (m *OverviewModel) getDisplayedTransactions() []model.Transaction {
	selectedItem := m.catList.SelectedItem()
	if selectedItem == nil {
		return nil
	}
	cat := selectedItem.(categoryItem)
	if cat.isSeparator {
		return nil
	}

	now := time.Now()
	var displayed []model.Transaction

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

		if !keep {
			continue
		}

		if cat.isUnreviewed {
			if !tx.IsReviewed {
				displayed = append(displayed, tx)
			}
		} else {
			if !tx.IsReviewed {
				// Skip unreviewed from regular categories
				continue
			}

			txCat := ""
			if tx.CategoryName != nil {
				txCat = *tx.CategoryName
			}

			if txCat == cat.fullPath || strings.HasPrefix(txCat, cat.fullPath+"/") {
				displayed = append(displayed, tx)
			}
		}
	}

	sort.Slice(displayed, func(i, j int) bool {
		return displayed[i].Date.After(displayed[j].Date)
	})

	return displayed
}

func (m *OverviewModel) updateTableData() {
	displayed := m.getDisplayedTransactions()
	var rows []table.Row

	for _, tx := range displayed {
		amountStr := fmt.Sprintf("%.2f %s", float64(tx.AmountCents)/100.0, tx.Currency)
		catName := ""
		if tx.CategoryName != nil {
			catName = *tx.CategoryName
		}
		rows = append(rows, table.Row{
			tx.Date.Format("2006-01-02"),
			tx.Payee,
			amountStr,
			catName,
		})
	}

	m.txTable.SetRows(rows)
}

func (m *OverviewModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	title := titleStyle.Render(" Finance Overview ")
	filterText := filterStyle.Render(fmt.Sprintf("Filter: %s (press tab to toggle)", m.filter.String()))
	header := lipgloss.JoinVertical(lipgloss.Left, title, filterText)

	var content string
	var help string

	var leftStyle, rightStyle lipgloss.Style
	if m.activePane == 0 {
		leftStyle = focusedPaneStyle
		rightStyle = blurredPaneStyle
	} else {
		leftStyle = blurredPaneStyle
		rightStyle = focusedPaneStyle
	}

	if !m.reviewingTx {
		listView := leftStyle.Render(m.catList.View())
		tableView := rightStyle.Render(m.txTable.View())
		panes := lipgloss.JoinHorizontal(lipgloss.Top, listView, tableView)
		help = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("\n[q] Quit • [tab] Toggle Filter • [h/l] Switch Panes • [enter] Edit Transaction • [j/k] Navigate")

		content = lipgloss.JoinVertical(lipgloss.Left, header, panes, help)
	} else {
		var formView string
		if m.reviewForm != nil {
			formView = m.reviewForm.View()
		}
		help = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("\n[esc] Abort • [/] Fuzzy Filter")
		content = lipgloss.JoinVertical(lipgloss.Left, header, formView, help)
	}

	return appStyle.Render(content)
}
