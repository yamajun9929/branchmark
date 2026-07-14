# VHS Tape for Branchmark Demo
# To record: ./generate-demo.sh

# Output configuration
Output demo.gif

# Settings (Compact and elegant terminal window - Optimized for GitHub README)
Set Shell "zsh"
Set FontSize 16
Set Width 850
Set Height 420
Set Theme "Catppuccin Mocha"
Set FontFamily "JetBrainsMono Nerd Font"
Set LetterSpacing 1.1
Set LineHeight 1.2
Set Padding 16
Set TypingSpeed 150ms

# Start brmk
Type "./brmk"
Enter
Sleep 1.5s

# 1. Expand "Repositories" folder to show icons
Down
Sleep 800ms
Right
Sleep 1s

# 2. Showcase quick Space tab-switching
Tab
Sleep 1.2s
Tab
Sleep 1.2s

# 3. Showcase super fast Tag Filtering (Search by "office")
Type "/"
Sleep 800ms
Type "office"
Sleep 1s
Enter
Sleep 2s

# 4. Clear Search to restore the full tree
Type "c"
Sleep 1.2s

# 5. Smoothly browse and expand remaining folders
# Move down to "Infrastructure"
Down
Sleep 600ms
Right
Sleep 800ms

# Move down to "Productivity" and expand
Down @600ms 3
Sleep 600ms
Right
Sleep 1.5s

# Gracefully quit
Type "q"
Sleep 500ms
