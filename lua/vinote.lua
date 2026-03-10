local M = {}

local binary = "vn"

--- Run vn command asynchronously and call on_result with parsed output.
---@param args string[]
---@param on_result fun(output: string)
local function vn_async(args, on_result)
  local stdout = {}
  vim.fn.jobstart(vim.list_extend({ binary }, args), {
    stdout_buffered = true,
    on_stdout = function(_, data)
      stdout = data
    end,
    on_exit = function(_, code)
      if code == 0 then
        local output = table.concat(stdout, "\n")
        vim.schedule(function()
          on_result(output)
        end)
      else
        vim.schedule(function()
          vim.notify("vn " .. table.concat(args, " ") .. " failed (exit " .. code .. ")", vim.log.levels.ERROR)
        end)
      end
    end,
  })
end

--- Get notes_dir from vn config (cached).
local notes_dir = nil
local function get_notes_dir(callback)
  if notes_dir then
    callback(notes_dir)
    return
  end
  -- Read from config or use default
  local home = vim.fn.expand("~")
  local config_path = home .. "/.config/vinote/config.toml"
  if vim.fn.filereadable(config_path) == 1 then
    local lines = vim.fn.readfile(config_path)
    for _, line in ipairs(lines) do
      local dir = line:match('^notes_dir%s*=%s*"(.-)"')
      if dir then
        notes_dir = dir:gsub("^~", home)
        callback(notes_dir)
        return
      end
    end
  end
  notes_dir = home .. "/notes"
  callback(notes_dir)
end

--- Open a note file from a Note JSON object.
---@param note table
local function open_note(note)
  get_notes_dir(function(dir)
    local path = dir .. "/" .. note.path .. ".md"
    vim.cmd.edit(path)
  end)
end

--- Open note picker (all notes, fuzzy search by title).
function M.open()
  vn_async({ "query", "--all", "--json" }, function(output)
    local notes = vim.json.decode(output) or {}
    local items = {}
    for _, note in ipairs(notes) do
      table.insert(items, {
        text = note.title,
        file = note.path .. ".md",
        note = note,
      })
    end

    get_notes_dir(function(dir)
      require("snacks.picker").pick({
        title = "Open Note",
        items = items,
        format = function(item)
          return { { item.text, "Normal" } }
        end,
        preview = function(ctx)
          local path = dir .. "/" .. ctx.item.file
          return require("snacks.picker.preview").file(ctx, { path = path })
        end,
        confirm = function(picker, item)
          picker:close()
          open_note(item.note)
        end,
      })
    end)
  end)
end

--- Search notes content (live grep via snacks).
function M.search()
  get_notes_dir(function(dir)
    require("snacks.picker").grep({ dirs = { dir } })
  end)
end

--- Topics picker (tag:topic, not archived).
function M.topics()
  vn_async({ "query", "--tag=topic", "--not=archived", "--json" }, function(output)
    local notes = vim.json.decode(output) or {}
    local items = {}
    for _, note in ipairs(notes) do
      table.insert(items, {
        text = note.title,
        file = note.path .. ".md",
        note = note,
      })
    end

    get_notes_dir(function(dir)
      require("snacks.picker").pick({
        title = "Topics",
        items = items,
        format = function(item)
          return { { item.text, "Normal" } }
        end,
        preview = function(ctx)
          local path = dir .. "/" .. ctx.item.file
          return require("snacks.picker.preview").file(ctx, { path = path })
        end,
        confirm = function(picker, item)
          picker:close()
          open_note(item.note)
        end,
      })
    end)
  end)
end

--- Backlinks picker for current buffer.
function M.backlinks()
  get_notes_dir(function(dir)
    local buf_path = vim.api.nvim_buf_get_name(0)
    -- Make path relative to notes_dir, strip .md
    local rel = buf_path:gsub("^" .. vim.pesc(dir) .. "/", ""):gsub("%.md$", "")

    vn_async({ "backlinks", rel }, function(output)
      local notes = vim.json.decode(output) or {}
      if #notes == 0 then
        vim.notify("No backlinks found", vim.log.levels.INFO)
        return
      end

      local items = {}
      for _, note in ipairs(notes) do
        table.insert(items, {
          text = note.title,
          file = note.path .. ".md",
          note = note,
        })
      end

      require("snacks.picker").pick({
        title = "Backlinks",
        items = items,
        format = function(item)
          return { { item.text, "Normal" } }
        end,
        preview = function(ctx)
          local path = dir .. "/" .. ctx.item.file
          return require("snacks.picker.preview").file(ctx, { path = path })
        end,
        confirm = function(picker, item)
          picker:close()
          open_note(item.note)
        end,
      })
    end)
  end)
end

--- Follow [[wikilink]] under cursor.
function M.follow_link()
  local line = vim.api.nvim_get_current_line()
  local col = vim.api.nvim_win_get_cursor(0)[2] + 1

  -- Find [[...]] surrounding cursor
  local link = nil
  for s, target, e in line:gmatch("()%[%[([^%]|]+)[^%]]*%]%]()") do
    if col >= s and col <= e then
      link = target
      break
    end
  end

  if not link then
    -- Fallback to default gf
    vim.cmd("normal! gf")
    return
  end

  vn_async({ "resolve", link }, function(output)
    local path = vim.trim(output)
    if path ~= "" then
      vim.cmd.edit(path)
    else
      vim.notify("Wikilink not found: " .. link, vim.log.levels.WARN)
    end
  end)
end

--- Weekly view in a float window.
function M.weekly()
  vn_async({ "weekly-view" }, function(output)
    local data = vim.json.decode(output)
    if not data then
      vim.notify("Failed to parse weekly data", vim.log.levels.ERROR)
      return
    end

    -- If weekly note doesn't exist, create it first
    if not data.file_exists then
      vn_async({ "weekly", "--create" }, function()
        -- Refresh the view
        M.weekly()
      end)
      return
    end

    -- Build buffer content
    local lines = {}
    table.insert(lines, "# " .. data.week .. " — " .. data.date_range)
    table.insert(lines, "")

    table.insert(lines, "## Meetings")
    if data.meetings and #data.meetings > 0 then
      for _, m in ipairs(data.meetings) do
        table.insert(lines, "  - " .. m.title)
      end
    else
      table.insert(lines, "  (none)")
    end

    table.insert(lines, "")
    table.insert(lines, "## Topics")
    if data.topics and #data.topics > 0 then
      for _, t in ipairs(data.topics) do
        table.insert(lines, "  - " .. t.title)
      end
    else
      table.insert(lines, "  (none)")
    end

    -- Collect selectable items by line number
    local selectable = {}
    local line_idx = 1
    for _, l in ipairs(lines) do
      local title = l:match("^  %- (.+)")
      if title then
        -- Find the matching note
        local all_notes = {}
        if data.meetings then vim.list_extend(all_notes, data.meetings) end
        if data.topics then vim.list_extend(all_notes, data.topics) end
        for _, n in ipairs(all_notes) do
          if n.title == title then
            selectable[line_idx] = n
            break
          end
        end
      end
      line_idx = line_idx + 1
    end

    -- Create float
    local buf = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
    vim.bo[buf].modifiable = false
    vim.bo[buf].buftype = "nofile"
    vim.bo[buf].filetype = "markdown"

    local width = math.max(60, math.floor(vim.o.columns * 0.5))
    local height = math.min(#lines + 2, math.floor(vim.o.lines * 0.7))
    local win = vim.api.nvim_open_win(buf, true, {
      relative = "editor",
      width = width,
      height = height,
      col = math.floor((vim.o.columns - width) / 2),
      row = math.floor((vim.o.lines - height) / 2),
      style = "minimal",
      border = "rounded",
      title = " Weekly View ",
      title_pos = "center",
    })

    -- Keymaps for the float
    local opts = { buffer = buf, nowait = true }

    vim.keymap.set("n", "q", function()
      vim.api.nvim_win_close(win, true)
    end, opts)

    vim.keymap.set("n", "<Esc>", function()
      vim.api.nvim_win_close(win, true)
    end, opts)

    vim.keymap.set("n", "<CR>", function()
      local cursor = vim.api.nvim_win_get_cursor(win)
      local note = selectable[cursor[1]]
      if note then
        vim.api.nvim_win_close(win, true)
        open_note(note)
      end
    end, opts)

    vim.keymap.set("n", "e", function()
      vim.api.nvim_win_close(win, true)
      get_notes_dir(function(dir)
        vim.cmd.edit(dir .. "/" .. data.file_path)
      end)
    end, opts)
  end)
end

--- Create a new note with frontmatter.
function M.new()
  get_notes_dir(function(dir)
    vim.ui.input({ prompt = "Note path (relative to notes): " }, function(input)
      if not input or input == "" then
        return
      end

      -- Ensure .md extension
      local rel_path = input
      if not rel_path:match("%.md$") then
        rel_path = rel_path .. ".md"
      end

      local abs_path = dir .. "/" .. rel_path
      local title = vim.fn.fnamemodify(rel_path, ":t:r")

      -- Create directory if needed
      local parent = vim.fn.fnamemodify(abs_path, ":h")
      vim.fn.mkdir(parent, "p")

      -- Write frontmatter
      local content = "---\ntitle: " .. title .. "\ntags:\n---\n\n"
      local f = io.open(abs_path, "w")
      if f then
        f:write(content)
        f:close()
        vim.cmd.edit(abs_path)
        -- Place cursor after frontmatter
        vim.api.nvim_win_set_cursor(0, { 6, 0 })
      else
        vim.notify("Failed to create: " .. abs_path, vim.log.levels.ERROR)
      end
    end)
  end)
end

--- Setup keybindings for markdown buffers.
function M.setup(opts)
  opts = opts or {}
  if opts.binary then
    binary = opts.binary
  end
  if opts.notes_dir then
    notes_dir = vim.fn.expand(opts.notes_dir)
  end

  local group = vim.api.nvim_create_augroup("vinote", { clear = true })
  vim.api.nvim_create_autocmd("FileType", {
    group = group,
    pattern = "markdown",
    callback = function(ev)
      local bopts = { buffer = ev.buf }
      vim.keymap.set("n", "<leader>vw", M.weekly, vim.tbl_extend("force", bopts, { desc = "Weekly view" }))
      vim.keymap.set("n", "<leader>vo", M.open, vim.tbl_extend("force", bopts, { desc = "Open note" }))
      vim.keymap.set("n", "<leader>vs", M.search, vim.tbl_extend("force", bopts, { desc = "Search notes" }))
      vim.keymap.set("n", "<leader>vn", M.new, vim.tbl_extend("force", bopts, { desc = "New note" }))
      vim.keymap.set("n", "<leader>vt", M.topics, vim.tbl_extend("force", bopts, { desc = "Topics" }))
      vim.keymap.set("n", "<leader>vb", M.backlinks, vim.tbl_extend("force", bopts, { desc = "Backlinks" }))
      vim.keymap.set("n", "gf", M.follow_link, vim.tbl_extend("force", bopts, { desc = "Follow wikilink" }))
    end,
  })
end

return M
