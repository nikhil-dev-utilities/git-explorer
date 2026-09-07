# The filter box is always focused, so every verb takes a modifier

Typing goes straight to the filter and the list narrows as you type, fzf-style, with no
key needed to begin. The core loop of this tool is "filter, then select everything that
matched", and this makes that loop as short as it can be.

The consequence is that every letter is text, so no verb can be a bare letter. The
rejected alternative was a modal filter — `/` to enter, Esc to leave — which preserves
`c` for clone and `a` for select-all and is what vim, less and lazygit users expect. It
was turned down for the keystroke it adds to the most common action in the tool.

## Consequences — read this before "fixing" the keymap

The keymap looks mnemonically poor on purpose. `^o` for select-all and `^y` for
switch-Host are not first choices; they are what remains after the collisions below.
**Do not rebind these to the obvious letters.** In particular:

- `^h` **is** backspace and `^i` **is** Tab. Binding either breaks text editing in the
  filter box. `^m` and `^[` are Enter and Esc.
- `^a`, `^e`, `^u`, `^w`, `^k` are readline home / end / kill. Users expect them to work
  *inside* the filter box, because it is a text box.
- `^c`, `^z`, `^d` belong to the terminal.

The ctrl bindings are the documented baseline because they work in a stock macOS
Terminal.app with no configuration. Mnemonic alt aliases (`alt-c`, `alt-a`, `alt-h`) are
additionally bound and cost a few lines; they are a bonus for terminals that send Meta,
never a requirement — neither Terminal.app nor iTerm2 sends Meta for Option by default.
