local map = vim.keymap.set

map("n", "<leader>w", "<cmd>write<cr>", { desc = "Guardar" })
map("n", "<leader>q", "<cmd>quit<cr>", { desc = "Salir" })
map("n", "<Esc>", "<cmd>nohlsearch<cr>")
