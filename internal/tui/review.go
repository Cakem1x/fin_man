package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/Cakem1x/fin_man/internal/categorize"
	"github.com/Cakem1x/fin_man/internal/importer"
	"github.com/Cakem1x/fin_man/internal/model"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ReviewAction string

const (
	ActionDiscard      ReviewAction = "Discard"
	ActionSave         ReviewAction = "Save changes (keep unreviewed)"
	ActionSaveReviewed ReviewAction = "Save & Mark Reviewed"
	ActionSplit        ReviewAction = "Split via Receipt Matcher"
)

type ReviewResult struct {
	Category string
	Tags     []string
	Memo     string
	Action   ReviewAction
	Splits   []model.Transaction
}

type ReviewFinishedMsg struct {
	Result *ReviewResult
}

type ReviewPane int

const (
	PaneTop ReviewPane = iota
	PaneCategory
	PaneTags
	PaneExit
)

type selectableItem struct {
	id       string
	title    string
	selected bool
	isMulti  bool
	desc     string
}

func (i selectableItem) Title() string {
	prefix := "  "
	if i.isMulti {
		if i.selected {
			prefix = "[x] "
		} else {
			prefix = "[ ] "
		}
	} else {
		if i.selected {
			prefix = "★ "
		}
	}
	return prefix + i.title
}
func (i selectableItem) Description() string { return i.desc }
func (i selectableItem) FilterValue() string { return i.title }

type ReviewModel struct {
	Tx model.Transaction

	catList  list.Model
	tagList  list.Model
	exitList list.Model

	memoInput   textinput.Model
	editingMemo bool

	addingNew     bool
	addingNewType string
	newInput      textinput.Model

	askingCSVPath bool
	filePicker    filepicker.Model
	statusMsg     string

	selectedCategory string
	selectedTags     map[string]bool
	memo             string

	focusedPane ReviewPane

	width  int
	height int
}

func NewReviewModel(tx model.Transaction, categories []string, tags []string, suggestion *categorize.Estimation, width, height int) ReviewModel {
	m := ReviewModel{
		Tx:               tx,
		selectedTags:     make(map[string]bool),
		memo:             tx.Memo,
		focusedPane:      PaneCategory,
		width:            width,
		height:           height,
	}

	var suggestedText string
	if suggestion != nil && suggestion.Confidence >= 0.5 {
		m.selectedCategory = suggestion.CategoryName
		suggestedText = fmt.Sprintf("Auto-suggested: %.0f%% confidence", suggestion.Confidence*100)
	}

	// Setup Category List
	catDel := list.NewDefaultDelegate()
	catDel.ShowDescription = true
	m.catList = list.New([]list.Item{}, catDel, 0, 0)
	m.catList.Title = "Category (Primary Budget Group)"
	m.catList.SetShowHelp(false)
	m.catList.SetShowStatusBar(false)
	m.catList.SetFilteringEnabled(true)

	var catItems []list.Item
	catItems = append(catItems, selectableItem{id: "", title: "None", selected: m.selectedCategory == "", isMulti: false})
	for _, c := range categories {
		desc := ""
		if suggestion != nil && suggestion.CategoryName == c {
			desc = suggestedText
		}
		catItems = append(catItems, selectableItem{id: c, title: c, selected: m.selectedCategory == c, isMulti: false, desc: desc})
	}
	m.catList.SetItems(catItems)

	// Setup Tag List
	tagDel := list.NewDefaultDelegate()
	tagDel.ShowDescription = false
	m.tagList = list.New([]list.Item{}, tagDel, 0, 0)
	m.tagList.Title = "Tags (Cross-category labels)"
	m.tagList.SetShowHelp(false)
	m.tagList.SetShowStatusBar(false)
	m.tagList.SetFilteringEnabled(true)

	var tagItems []list.Item
	for _, t := range tags {
		tagItems = append(tagItems, selectableItem{id: t, title: t, selected: false, isMulti: true})
	}
	m.tagList.SetItems(tagItems)

	// Setup Exit List
	exitDel := list.NewDefaultDelegate()
	exitDel.ShowDescription = false

	saveReviewedTitle := "Save & Mark Reviewed"
	saveUnreviewedTitle := "Save changes (keep unreviewed)"

	if tx.IsReviewed {
		saveReviewedTitle = "Save changes (keep reviewed)"
		saveUnreviewedTitle = "Save & Mark Unreviewed"
	}

	m.exitList = list.New([]list.Item{
		selectableItem{id: string(ActionSaveReviewed), title: saveReviewedTitle, isMulti: false},
		selectableItem{id: string(ActionSave), title: saveUnreviewedTitle, isMulti: false},
		selectableItem{id: string(ActionDiscard), title: "Discard", isMulti: false},
	}, exitDel, 0, 0)
	m.exitList.Title = "Save Changes?"
	m.exitList.SetShowHelp(false)
	m.exitList.SetShowStatusBar(false)
	m.exitList.SetFilteringEnabled(false)

	// Setup Memo Input
	m.memoInput = textinput.New()
	m.memoInput.Placeholder = "Enter memo..."
	m.memoInput.SetValue(m.memo)

	// Setup New Input
	m.newInput = textinput.New()
	m.newInput.Width = 30

	fp := filepicker.New()
	fp.AllowedTypes = []string{".csv"}
	fp.CurrentDirectory, _ = os.Getwd()
	fp.SetHeight(10)
	m.filePicker = fp

	m.updateSizes()
	m.updateFocus()
	return m
}

func (m *ReviewModel) updateSizes() {
	halfWidth := m.width/2 - 4
	listHeight := m.height/2 - 2
	if halfWidth < 10 {
		halfWidth = 10
	}
	if listHeight < 5 {
		listHeight = 5
	}
	m.catList.SetSize(halfWidth, listHeight)
	m.tagList.SetSize(halfWidth, listHeight)
	m.exitList.SetSize(40, 10)
	m.memoInput.Width = m.width - 6
}

func (m *ReviewModel) updateFocus() {
	if m.focusedPane == PaneCategory {
		m.catList.Styles.Title = m.catList.Styles.Title.Background(lipgloss.Color("62"))
	} else {
		m.catList.Styles.Title = m.catList.Styles.Title.Background(lipgloss.Color("240"))
	}
	if m.focusedPane == PaneTags {
		m.tagList.Styles.Title = m.tagList.Styles.Title.Background(lipgloss.Color("62"))
	} else {
		m.tagList.Styles.Title = m.tagList.Styles.Title.Background(lipgloss.Color("240"))
	}
}

func (m *ReviewModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.filePicker.Init())
}

func (m *ReviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if m.askingCSVPath {
		var cmd tea.Cmd
		m.filePicker, cmd = m.filePicker.Update(msg)

		if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
			matcher, err := importer.NewAmazonPrivacyCSVMatcher(path)
			if err != nil {
				m.statusMsg = fmt.Sprintf("Error loading CSV: %v", err)
				m.askingCSVPath = false
				return m, nil
			}
			items, matched, err := matcher.Match(m.Tx.Memo, m.Tx.Date)
			if err != nil || !matched {
				m.statusMsg = "No matching items found in CSV."
				m.askingCSVPath = false
				return m, nil
			}

			var splits []model.Transaction
			var sum int64
			for i, it := range items {
				splits = append(splits, model.Transaction{
					ID:              fmt.Sprintf("%s-split-%d", m.Tx.ID, i),
					Date:            m.Tx.Date,
					Payee:           m.Tx.Payee,
					AmountCents:     it.AmountCents,
					Currency:        m.Tx.Currency,
					Memo:            it.Title,
					IsReviewed:      false,
					ArchiveFilePath: m.Tx.ArchiveFilePath,
				})
				sum += it.AmountCents
			}

			if sum != m.Tx.AmountCents {
				splits = append(splits, model.Transaction{
					ID:              fmt.Sprintf("%s-split-remainder", m.Tx.ID),
					Date:            m.Tx.Date,
					Payee:           m.Tx.Payee,
					AmountCents:     m.Tx.AmountCents - sum,
					Currency:        m.Tx.Currency,
					Memo:            "Shipping / Tax / Remainder",
					IsReviewed:      false,
					ArchiveFilePath: m.Tx.ArchiveFilePath,
				})
			}

			return m, func() tea.Msg {
				return ReviewFinishedMsg{
					Result: &ReviewResult{
						Action: ActionSplit,
						Splits: splits,
					},
				}
			}
		}

		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			m.askingCSVPath = false
			return m, nil
		}

		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()
	case tea.KeyMsg:
		if m.focusedPane == PaneExit {
			switch msg.String() {
			case "esc":
				m.focusedPane = PaneCategory
				m.updateFocus()
				return m, nil
			case "enter":
				sel := m.exitList.SelectedItem().(selectableItem)
				return m, func() tea.Msg {
					tags := []string{}
					for t, selected := range m.selectedTags {
						if selected {
							tags = append(tags, t)
						}
					}
					return ReviewFinishedMsg{
						Result: &ReviewResult{
							Category: m.selectedCategory,
							Tags:     tags,
							Memo:     m.memo,
							Action:   ReviewAction(sel.id),
						},
					}
				}
			}
			m.exitList, cmd = m.exitList.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		if m.addingNew {
			switch msg.String() {
			case "esc":
				m.addingNew = false
				m.newInput.Blur()
				return m, nil
			case "enter":
				val := strings.TrimSpace(m.newInput.Value())
				if val != "" {
					if m.addingNewType == "category" {
						items := m.catList.Items()
						items = append(items, selectableItem{id: val, title: val, selected: true, isMulti: false})
						m.selectedCategory = val
						for i, it := range items {
							s := it.(selectableItem)
							s.selected = s.id == m.selectedCategory
							items[i] = s
						}
						m.catList.SetItems(items)
					} else {
						items := m.tagList.Items()
						items = append(items, selectableItem{id: val, title: val, selected: true, isMulti: true})
						m.selectedTags[val] = true
						m.tagList.SetItems(items)
					}
				}
				m.addingNew = false
				m.newInput.Blur()
				return m, nil
			}
			m.newInput, cmd = m.newInput.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}


		if m.editingMemo {
			switch msg.String() {
			case "esc", "enter":
				m.editingMemo = false
				m.memo = m.memoInput.Value()
				m.memoInput.Blur()
				return m, nil
			}
			m.memoInput, cmd = m.memoInput.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		if m.focusedPane == PaneTop {
			switch msg.String() {
			case "enter":
				m.editingMemo = true
				m.memoInput.Focus()
				return m, textinput.Blink
			}
		}

		switch msg.String() {
		case "m":
			m.askingCSVPath = true
			m.filePicker.CurrentDirectory, _ = os.Getwd()
			return m, m.filePicker.Init()
		case "n":
			if m.focusedPane == PaneCategory && m.catList.FilterState() != list.Filtering {
				m.addingNew = true
				m.addingNewType = "category"
				m.newInput.Placeholder = "New Category Name"
				m.newInput.Reset()
				m.newInput.Focus()
				return m, textinput.Blink
			} else if m.focusedPane == PaneTags && m.tagList.FilterState() != list.Filtering {
				m.addingNew = true
				m.addingNewType = "tags"
				m.newInput.Placeholder = "New Tag Name"
				m.newInput.Reset()
				m.newInput.Focus()
				return m, textinput.Blink
			}
		case "esc":
			if m.catList.FilterState() == list.Filtering {
				m.catList.ResetFilter()
				return m, nil
			}
			if m.tagList.FilterState() == list.Filtering {
				m.tagList.ResetFilter()
				return m, nil
			}
			m.focusedPane = PaneExit
			m.updateFocus()
			return m, nil
		case "tab":
			switch m.focusedPane {
			case PaneTop:
				m.focusedPane = PaneCategory
			case PaneCategory:
				m.focusedPane = PaneTags
			case PaneTags:
				m.focusedPane = PaneTop
			}
			m.updateFocus()
			return m, nil
		case "shift+tab":
			switch m.focusedPane {
			case PaneTop:
				m.focusedPane = PaneTags
			case PaneCategory:
				m.focusedPane = PaneTop
			case PaneTags:
				m.focusedPane = PaneCategory
			}
			m.updateFocus()
			return m, nil
		case "enter":
			if m.focusedPane == PaneCategory && m.catList.FilterState() != list.Filtering {
				if sel := m.catList.SelectedItem(); sel != nil {
					si := sel.(selectableItem)
					m.selectedCategory = si.id
					items := m.catList.Items()
					for i, it := range items {
						s := it.(selectableItem)
						s.selected = s.id == m.selectedCategory
						items[i] = s
					}
					m.catList.SetItems(items)
				}
			} else if m.focusedPane == PaneTags && m.tagList.FilterState() != list.Filtering {
				if sel := m.tagList.SelectedItem(); sel != nil {
					si := sel.(selectableItem)
					m.selectedTags[si.id] = !m.selectedTags[si.id]
					items := m.tagList.Items()
					for i, it := range items {
						s := it.(selectableItem)
						if s.id == si.id {
							s.selected = m.selectedTags[s.id]
							items[i] = s
						}
					}
					m.tagList.SetItems(items)
				}
			}
		}

		switch m.focusedPane {
		case PaneCategory:
			m.catList, cmd = m.catList.Update(msg)
			cmds = append(cmds, cmd)
		case PaneTags:
			m.tagList, cmd = m.tagList.Update(msg)
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *ReviewModel) View() string {
	if m.addingNew {
		dialogBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Render("Enter new " + m.addingNewType + ":\n\n" + m.newInput.View())

		overlay := lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			dialogBox,
		)
		return overlay
	}

	if m.askingCSVPath {
		dialogBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Render("Select Amazon Privacy CSV File:\n\n" + m.filePicker.View())

		overlay := lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			dialogBox,
		)
		return overlay
	}

	if m.focusedPane == PaneExit {
		dialogBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Render(m.exitList.View())

		overlay := lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			dialogBox,
		)
		return overlay
	}

	topStyle := blurredPaneStyle
	if m.focusedPane == PaneTop {
		topStyle = focusedPaneStyle
	}

	var topContent string
	if m.editingMemo {
		topContent = "Editing Memo:\n\n" + m.memoInput.View()
	} else {
		buttonStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		if m.focusedPane == PaneTop {
			buttonStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("62")).Padding(0, 1)
		}

		status := ""
		if m.statusMsg != "" {
			status = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(m.statusMsg)
		}

		topContent = fmt.Sprintf(
			"Date:   %s\nPayee:  %s\nAmount: %.2f %s\nMemo:   %s%s\n\n%s",
			m.Tx.Date.Format("2006-01-02"),
			m.Tx.Payee,
			float64(m.Tx.AmountCents)/100.0,
			m.Tx.Currency,
			m.memo,
			status,
			buttonStyle.Render("[ Edit Memo ]"),
		)
	}

	topView := topStyle.Width(m.width - 4).Render(topContent)

	catStyle := blurredPaneStyle
	if m.focusedPane == PaneCategory {
		catStyle = focusedPaneStyle
	}
	tagStyle := blurredPaneStyle
	if m.focusedPane == PaneTags {
		tagStyle = focusedPaneStyle
	}

	bottomLeft := catStyle.Width(m.width/2 - 3).Render(m.catList.View())
	bottomRight := tagStyle.Width(m.width/2 - 3).Render(m.tagList.View())

	bottom := lipgloss.JoinHorizontal(lipgloss.Top, bottomLeft, bottomRight)
	formView := lipgloss.JoinVertical(lipgloss.Left, topView, bottom)

	helpText := "\n[tab] Switch Pane • [enter] Select/Action • [esc] Exit Options • [/] Filter • [m] Receipt Matcher"
	if m.editingMemo {
		helpText = "\n[enter/esc] Done Editing"
	}
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(helpText)

	return lipgloss.JoinVertical(lipgloss.Left, formView, help)
}
