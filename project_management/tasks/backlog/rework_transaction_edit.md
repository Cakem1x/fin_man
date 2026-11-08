Fully re-work TUI UX. Main focus is the transaction editing, but other places need to be touched so we get consistent UX.

What's good:
- Classification is nice. Ordering by classification is very useful when similar transaction were seen. Allows user to very quickly pick the correct category and tag.
- the 'diff' view when exiting is great

What's bad:
- Keybinding and UI are inconsistent and get in each other's way (e.g. when typing in a term for filtering, if that term includes 'n', the new category dialog gets opened)
- Fuzzy filter/search features are not in place or do not work properly

# Proposal
## Full UI Changes
- Adapt UX to make use of left-to-right panel transitions.
- With 'enter', transition one level down (meaning to the right panel). With 'esc', transition one level up (to the left panel).
- Each view usually displays the focused panel with large size and the next panel with small size. Size ratio is golden ratio. This is a general guideline and can be broken, if a specific arrangement requires it. But the general UI flow should follow this.
- Apply to existing views:
  - Starting view shows categories in the left panel (large), including the stats that we already have implemented.
  - On the right side (small), show transactions of that category.
  - Pressing enter should fix the highlighted category, make the left panel small and transition cursor to the right panel (make it large).
  - Now, transactions of the chosen category can be navigated. Pressing esc here will transition back to previous view. Pressing enter here will transition into the edit transaction menu (see below). I think this should become a new fullscreen view, seeing other transactions of the same category on the left side is not useful in this case and wastes space.

## Edit Transactions
When editing transactions, make more use of the top panel that shows the transaction state. Base new UX on this view. Enrich this view with the diff view, always showing the changes. Enhance diff view rendering to use colors that highlight what was changed (added/removed). Especially highlight when new tags or categories are added
- Move the button to call the Receipt Matcher feature into that view, on the same conceptual level as the edit memo button.
- Avoid nested keybinds, i.e. different keybindings in different panels / views. Especially do not let them conflict with each other!
- Let the diff view be navigatable. Picking (and adding) tags and categories should be accessible not via two panels below, but by moving curser over 'tags' or 'category'. Show tag or category data on the right side (small, use golden ratio). Pressing enter makes right side panel larger and allows changing assigned tags/categories. enter or esc brings view back to left panel.

## Implementation Rules
I've sketched a rough idea of how the UX should look like. You need to ensure that UX is consistent over all views. Ask questions if necessary to change behavior for consistency. Also keep in mind how much extra custom code is needed. It is worth falling back to slightly different behaviors if that is better supported by UI libs we use.
