# Changelog

## [0.4.0](https://github.com/nikhil-dev-utilities/git-explorer/compare/v0.3.1...v0.4.0) (2026-09-09)


### Features

* **logging:** wire up real log statements, default path to ~/.logs/git-explorer ([#82](https://github.com/nikhil-dev-utilities/git-explorer/issues/82)) ([61cf364](https://github.com/nikhil-dev-utilities/git-explorer/commit/61cf364032f4effb0e3066d8bf6b11162dddbccf)), closes [#80](https://github.com/nikhil-dev-utilities/git-explorer/issues/80)


### Bug Fixes

* **github:** paginate ListRepos, fetchMemberships, fetchCollaboratorOrgs ([#86](https://github.com/nikhil-dev-utilities/git-explorer/issues/86)) ([1872541](https://github.com/nikhil-dev-utilities/git-explorer/commit/1872541354089fba506144a1014495c92b0966c7)), closes [#83](https://github.com/nikhil-dev-utilities/git-explorer/issues/83)
* **tui:** give the Org pane a real ^s sort (name ⇄ Affiliation) ([#90](https://github.com/nikhil-dev-utilities/git-explorer/issues/90)) ([f65dfa6](https://github.com/nikhil-dev-utilities/git-explorer/commit/f65dfa68629e29bfe98c6487b0778a3e208b4fea)), closes [#88](https://github.com/nikhil-dev-utilities/git-explorer/issues/88)
* **tui:** label the Org/Repo pane's quick-filter line ([#89](https://github.com/nikhil-dev-utilities/git-explorer/issues/89)) ([3f31d34](https://github.com/nikhil-dev-utilities/git-explorer/commit/3f31d3468a1352bf3a6b3203d02a62f0068489fc)), closes [#87](https://github.com/nikhil-dev-utilities/git-explorer/issues/87)

## [0.3.1](https://github.com/nikhil-dev-utilities/git-explorer/compare/v0.3.0...v0.3.1) (2026-09-09)


### Bug Fixes

* **tui:** window the Org/Repo pane lists so the cursor stays visible ([#78](https://github.com/nikhil-dev-utilities/git-explorer/issues/78)) ([a44c37e](https://github.com/nikhil-dev-utilities/git-explorer/commit/a44c37e00fc8173dcc80e9dc37dc027dd46d8bfa))

## [0.3.0](https://github.com/nikhil-dev-utilities/git-explorer/compare/v0.2.0...v0.3.0) (2026-09-09)


### Features

* auto-discover gh-authenticated host(s) for zero-config startup ([#71](https://github.com/nikhil-dev-utilities/git-explorer/issues/71)) ([de18187](https://github.com/nikhil-dev-utilities/git-explorer/commit/de18187d56c8ec79470a49e60a05b74902adb6db)), closes [#65](https://github.com/nikhil-dev-utilities/git-explorer/issues/65)
* **cmd:** bootstrap config dir, starter config.yaml, and log location ([#69](https://github.com/nikhil-dev-utilities/git-explorer/issues/69)) ([4de0061](https://github.com/nikhil-dev-utilities/git-explorer/commit/4de0061f8f2654d94f62496e58d8a3c6e50fe548)), closes [#64](https://github.com/nikhil-dev-utilities/git-explorer/issues/64)
* **tui:** render modal dialogs' actions as button-style widgets ([#74](https://github.com/nikhil-dev-utilities/git-explorer/issues/74)) ([06a1359](https://github.com/nikhil-dev-utilities/git-explorer/commit/06a1359fe3acfa31985b83d9a88604a43aa5c2e9)), closes [#68](https://github.com/nikhil-dev-utilities/git-explorer/issues/68)
* **tui:** soften "not authenticated" from Fatal to pane-scoped for discovered hosts ([#73](https://github.com/nikhil-dev-utilities/git-explorer/issues/73)) ([932d757](https://github.com/nikhil-dev-utilities/git-explorer/commit/932d757210abf3540f3e47cd43450dc5763a5e18)), closes [#67](https://github.com/nikhil-dev-utilities/git-explorer/issues/67)
* **tui:** split Browse's footer into a status line and a key-hint grid ([#72](https://github.com/nikhil-dev-utilities/git-explorer/issues/72)) ([afb22f4](https://github.com/nikhil-dev-utilities/git-explorer/commit/afb22f4c228a56c50c8a337b0283b41bb99a8785)), closes [#66](https://github.com/nikhil-dev-utilities/git-explorer/issues/66)


### Bug Fixes

* **tui:** trim withBrowseFooter's trailing newline, fixing missing top border ([#75](https://github.com/nikhil-dev-utilities/git-explorer/issues/75)) ([fba1944](https://github.com/nikhil-dev-utilities/git-explorer/commit/fba194440e9a3fc4febc8c2a1ff78ae95c45ff61))

## [0.2.0](https://github.com/nikhil-dev-utilities/git-explorer/compare/v0.1.1...v0.2.0) (2026-09-08)


### Features

* **tui:** add a bordered, focus-colored bounding box to each pane ([#63](https://github.com/nikhil-dev-utilities/git-explorer/issues/63)) ([ccb2b32](https://github.com/nikhil-dev-utilities/git-explorer/commit/ccb2b32a5335a26da7e86177f8adefceb28e0d8c))
* **tui:** add Left/Right as additive Orgs&lt;-&gt;Repos pane navigation ([#61](https://github.com/nikhil-dev-utilities/git-explorer/issues/61)) ([fac0431](https://github.com/nikhil-dev-utilities/git-explorer/commit/fac04317cf126f321f5343e8b2b5651d079ad8a3))
* **tui:** add persistent footer to Browse and HostSwitch ([#60](https://github.com/nikhil-dev-utilities/git-explorer/issues/60)) ([00593a5](https://github.com/nikhil-dev-utilities/git-explorer/commit/00593a534e24e3a950bf1e305dfcc5af29157237))
* **tui:** make the clone dialog's target path editable ([#62](https://github.com/nikhil-dev-utilities/git-explorer/issues/62)) ([b535b40](https://github.com/nikhil-dev-utilities/git-explorer/commit/b535b401075113fbd000792a921b70bcd9ff597a))


### Bug Fixes

* **forge/github:** force -X GET on every gh api call ([#58](https://github.com/nikhil-dev-utilities/git-explorer/issues/58)) ([450b397](https://github.com/nikhil-dev-utilities/git-explorer/commit/450b397a0848452ef740faa7cc2bd6e2aa2f0a7a)), closes [#57](https://github.com/nikhil-dev-utilities/git-explorer/issues/57)

## [0.1.1](https://github.com/nikhil-dev-utilities/git-explorer/compare/v0.1.0...v0.1.1) (2026-09-07)


### Bug Fixes

* **ci:** run GoReleaser in the same workflow run as release-please ([#54](https://github.com/nikhil-dev-utilities/git-explorer/issues/54)) ([0953aab](https://github.com/nikhil-dev-utilities/git-explorer/commit/0953aab17ec82651796cd676ca1479b95c04063e))

## 0.1.0 (2026-09-07)


### Features

* **clone:** batch Run — bounded parallelism, never aborts on failure ([#23](https://github.com/nikhil-dev-utilities/git-explorer/issues/23)) ([303b93b](https://github.com/nikhil-dev-utilities/git-explorer/commit/303b93bce42263729ee883e5898f3bd43f9588f6)), closes [#19](https://github.com/nikhil-dev-utilities/git-explorer/issues/19)
* **clone:** cancellation — stop dispatch, terminate in-flight, keep results ([#24](https://github.com/nikhil-dev-utilities/git-explorer/issues/24)) ([aa6d04b](https://github.com/nikhil-dev-utilities/git-explorer/commit/aa6d04b264fbc930b48ae7575b2c8f5dbcbd0995)), closes [#20](https://github.com/nikhil-dev-utilities/git-explorer/issues/20)
* **clone:** classify + clone the happy path for one Repo ([#21](https://github.com/nikhil-dev-utilities/git-explorer/issues/21)) ([95468b8](https://github.com/nikhil-dev-utilities/git-explorer/commit/95468b80c7906c93fbff02ae31bba14a6d0a4ef4)), closes [#17](https://github.com/nikhil-dev-utilities/git-explorer/issues/17)
* **clone:** classify Skipped (protocol-normalized) and Conflict ([#22](https://github.com/nikhil-dev-utilities/git-explorer/issues/22)) ([bb2a0c7](https://github.com/nikhil-dev-utilities/git-explorer/commit/bb2a0c77109868f40b66dff211b54a08990a1b93)), closes [#18](https://github.com/nikhil-dev-utilities/git-explorer/issues/18)
* **cmd:** add composition root wiring Forge, Config, and Clone into the TUI ([#50](https://github.com/nikhil-dev-utilities/git-explorer/issues/50)) ([270a177](https://github.com/nikhil-dev-utilities/git-explorer/commit/270a177e0493eaa4af727879939ff85ffb3e1516)), closes [#45](https://github.com/nikhil-dev-utilities/git-explorer/issues/45)
* **config:** add Config schema and Load() core with zero-config defaults ([#13](https://github.com/nikhil-dev-utilities/git-explorer/issues/13)) ([2c4c368](https://github.com/nikhil-dev-utilities/git-explorer/commit/2c4c3689133a0321d97d7f89b50cafb57c89ee45))
* **config:** logging bootstrap — slog + lumberjack, off-level no-op ([#16](https://github.com/nikhil-dev-utilities/git-explorer/issues/16)) ([7bf36e1](https://github.com/nikhil-dev-utilities/git-explorer/commit/7bf36e1ae210d4d0dc31066de4158e44df90818c)), closes [#12](https://github.com/nikhil-dev-utilities/git-explorer/issues/12)
* **config:** resolve per-Host default_target, name the file on parse errors ([#14](https://github.com/nikhil-dev-utilities/git-explorer/issues/14)) ([2693234](https://github.com/nikhil-dev-utilities/git-explorer/commit/2693234e9c4b7ecae32d73e26740c3e30d28f165)), closes [#10](https://github.com/nikhil-dev-utilities/git-explorer/issues/10)
* **forge/github:** add the command-runner seam ([1a71ea8](https://github.com/nikhil-dev-utilities/git-explorer/commit/1a71ea8dc2ead999d3946d55b6f0ef42605e6270))
* **forge/github:** check gh authentication before listing ([5bbf196](https://github.com/nikhil-dev-utilities/git-explorer/commit/5bbf19668a33017ad58692307f05928fb3554674))
* **forge/github:** implement ListRepos and CloneURL ([89f5622](https://github.com/nikhil-dev-utilities/git-explorer/commit/89f5622c074d7932a22f22d5a8c89cbdc224214d)), closes [#8](https://github.com/nikhil-dev-utilities/git-explorer/issues/8)
* **forge/github:** list Orgs on a Private Host ([caf20e5](https://github.com/nikhil-dev-utilities/git-explorer/commit/caf20e5a8a0e612bda50948ca95645cf87d15389)), closes [#6](https://github.com/nikhil-dev-utilities/git-explorer/issues/6)
* **forge/github:** list Orgs on a Public Host via member/collaborator union ([a4392b2](https://github.com/nikhil-dev-utilities/git-explorer/commit/a4392b282f5a9be1293050e84aa323796614cdb4))
* **forge/github:** stream Org pages on a Private Host, classify rate limits ([b21af43](https://github.com/nikhil-dev-utilities/git-explorer/commit/b21af43a644ebd00db5803b955f41ee30e1446e4)), closes [#7](https://github.com/nikhil-dev-utilities/git-explorer/issues/7)
* **forge/github:** wire ListOrgs to the Public Host path ([575345f](https://github.com/nikhil-dev-utilities/git-explorer/commit/575345f42b7abf25d190bcd0a1153d4338f9dc8f)), closes [#5](https://github.com/nikhil-dev-utilities/git-explorer/issues/5)
* **forge:** define port types and the Forge interface ([cd79940](https://github.com/nikhil-dev-utilities/git-explorer/commit/cd79940366b5e6ee1f7c754743b8c0ca326f9c52))
* **tui:** add DESIGN.md's mnemonic alt-key aliases ([#51](https://github.com/nikhil-dev-utilities/git-explorer/issues/51)) ([45162ea](https://github.com/nikhil-dev-utilities/git-explorer/commit/45162ea41f9ddf7971a2bfb87533dcd42500e44f)), closes [#46](https://github.com/nikhil-dev-utilities/git-explorer/issues/46)
* **tui:** clone dialog — path preview + org-subdirectory toggle ([#40](https://github.com/nikhil-dev-utilities/git-explorer/issues/40)) ([8066f1f](https://github.com/nikhil-dev-utilities/git-explorer/commit/8066f1f75676a217c4d10d326c751e5224ab904a)), closes [#31](https://github.com/nikhil-dev-utilities/git-explorer/issues/31)
* **tui:** clone run — modal progress + cancel + summary + retry ([#41](https://github.com/nikhil-dev-utilities/git-explorer/issues/41)) ([aeb6411](https://github.com/nikhil-dev-utilities/git-explorer/commit/aeb6411a00377a88742d09d4548d9fd8193e5627)), closes [#32](https://github.com/nikhil-dev-utilities/git-explorer/issues/32)
* **tui:** descend to Repos — two-pane navigation, filter/facets/sort ([#35](https://github.com/nikhil-dev-utilities/git-explorer/issues/35)) ([eba7729](https://github.com/nikhil-dev-utilities/git-explorer/commit/eba7729f6616587a1096fdcf1f17c53d535bae71)), closes [#26](https://github.com/nikhil-dev-utilities/git-explorer/issues/26)
* **tui:** failure surfaces — Fatal / pane-scoped / transient, distinct empty states ([#38](https://github.com/nikhil-dev-utilities/git-explorer/issues/38)) ([15ddbd4](https://github.com/nikhil-dev-utilities/git-explorer/commit/15ddbd47ce5000ecbc2e3f0296c94d22a062e25c)), closes [#29](https://github.com/nikhil-dev-utilities/git-explorer/issues/29)
* **tui:** foundation — Bubble Tea shell, browse Orgs with live filter ([#34](https://github.com/nikhil-dev-utilities/git-explorer/issues/34)) ([6acca17](https://github.com/nikhil-dev-utilities/git-explorer/commit/6acca173fb8b3efee6b8e0777dfd05b682ba7373))
* **tui:** help screen + keymap regression guard ([#42](https://github.com/nikhil-dev-utilities/git-explorer/issues/42)) ([a2a00ca](https://github.com/nikhil-dev-utilities/git-explorer/commit/a2a00cab387dc97b28a205ce8e4483398ea208b0)), closes [#33](https://github.com/nikhil-dev-utilities/git-explorer/issues/33)
* **tui:** host switching ([#37](https://github.com/nikhil-dev-utilities/git-explorer/issues/37)) ([39e20c7](https://github.com/nikhil-dev-utilities/git-explorer/commit/39e20c79c7bf2878648518c81e1de6c0e6308411)), closes [#28](https://github.com/nikhil-dev-utilities/git-explorer/issues/28)
* **tui:** narrow-terminal responsive layout, real two-column composition ([#39](https://github.com/nikhil-dev-utilities/git-explorer/issues/39)) ([a56f0e4](https://github.com/nikhil-dev-utilities/git-explorer/commit/a56f0e44170ef663e30a2680f35a9127a755aba9)), closes [#30](https://github.com/nikhil-dev-utilities/git-explorer/issues/30)
* **tui:** selection — Tab tick, select-all-matching, LeavePrompt ([#36](https://github.com/nikhil-dev-utilities/git-explorer/issues/36)) ([df63b74](https://github.com/nikhil-dev-utilities/git-explorer/commit/df63b74095678f1e1c68d170f642b6f01fb9a01a)), closes [#27](https://github.com/nikhil-dev-utilities/git-explorer/issues/27)


### Bug Fixes

* **forge/github:** stop depending on the real gh binary in runner tests ([f0ce469](https://github.com/nikhil-dev-utilities/git-explorer/commit/f0ce469688dec2f2248212886f46d232eb43d354))
* **forge:** carry Host on Org ([7111205](https://github.com/nikhil-dev-utilities/git-explorer/commit/7111205054a36cc2bbb2c107310e4f6eaf99bd43))
