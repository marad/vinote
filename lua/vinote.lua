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

--- Build picker items from notes JSON, with absolute file paths.
---@param notes table[]
---@param dir string
---@return table[]
local function notes_to_items(notes, dir)
  local items = {}
  for _, note in ipairs(notes) do
    table.insert(items, {
      text = note.path,
      file = dir .. "/" .. note.path .. ".md",
      note = note,
    })
  end
  return items
end

--- Format picker items with left-truncated paths.
---@param item snacks.picker.Item
---@param picker snacks.Picker
---@return snacks.picker.Highlight[]
local function format_note(item, picker)
  local ret = {}
  ret[#ret + 1] = {
    "",
    resolve = function(max_width)
      local path = item.text or ""
      local tw = vim.api.nvim_strwidth(path)
      if tw > max_width then
        path = "…" .. vim.fn.strcharpart(path, tw - max_width + 1, max_width - 1)
      end
      return { { path, "SnacksPickerFile" } }
    end,
  }
  return ret
end

--- Open a snacks picker for notes.
---@param title string
---@param items table[]
---@param dir string notes_dir absolute path
local function notes_picker(title, items, dir)
  Snacks.picker({
    title = title,
    items = items,
    cwd = dir,
    format = format_note,
    preview = "file",
    confirm = "jump",
  })
end

--- Open note picker (all notes, fuzzy search by title).
function M.open()
  vn_async({ "query", "--all", "--json" }, function(output)
    local notes = vim.json.decode(output) or {}
    get_notes_dir(function(dir)
      notes_picker("Open Note", notes_to_items(notes, dir), dir)
    end)
  end)
end

--- Search notes content (live grep via snacks).
function M.search()
  get_notes_dir(function(dir)
    Snacks.picker.grep({ dirs = { dir } })
  end)
end

--- Topics picker (tag:topic, not archived).
function M.topics()
  vn_async({ "query", "--tag=topic", "--not=archived", "--json" }, function(output)
    local notes = vim.json.decode(output) or {}
    get_notes_dir(function(dir)
      notes_picker("Topics", notes_to_items(notes, dir), dir)
    end)
  end)
end

--- Backlinks picker for current buffer.
function M.backlinks()
  get_notes_dir(function(dir)
    local buf_path = vim.api.nvim_buf_get_name(0)
    local rel = buf_path:gsub("^" .. vim.pesc(dir) .. "/", ""):gsub("%.md$", "")

    vn_async({ "backlinks", rel }, function(output)
      local notes = vim.json.decode(output) or {}
      if #notes == 0 then
        vim.notify("No backlinks found", vim.log.levels.INFO)
        return
      end
      notes_picker("Backlinks", notes_to_items(notes, dir), dir)
    end)
  end)
end

--- Follow [[wikilink]] under cursor.
function M.follow_link()
  local line = vim.api.nvim_get_current_line()
  local col = vim.api.nvim_win_get_cursor(0)[2] + 1

  local link = nil
  for s, target, e in line:gmatch("()%[%[([^%]|]+)[^%]]*%]%]()") do
    if col >= s and col <= e then
      link = target
      break
    end
  end

  if not link then
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
---@param week string|nil Week in YYYY-Www format (nil = current week)
function M.weekly(week)
  local args = { "weekly-view" }
  if week then
    table.insert(args, "--week=" .. week)
  end
  vn_async(args, function(output)
    local data = vim.json.decode(output)
    if not data then
      vim.notify("Failed to parse weekly data", vim.log.levels.ERROR)
      return
    end

    -- Show float regardless of file existence

    local lines = {}
    table.insert(lines, "# " .. data.week .. " — " .. data.date_range)
    table.insert(lines, "")

    local meetings = type(data.meetings) == "table" and data.meetings or {}
    local topics = type(data.topics) == "table" and data.topics or {}

    table.insert(lines, "## Topics")
    if #topics > 0 then
      for _, t in ipairs(topics) do
        table.insert(lines, "  - " .. t.title)
      end
    else
      table.insert(lines, "  (none)")
    end

    local day_names = { "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday" }

    table.insert(lines, "")
    table.insert(lines, "## Meetings")
    if #meetings > 0 then
      -- Group by date, sort by hour within each day
      local by_date = {}
      local dates = {}
      for _, m in ipairs(meetings) do
        local date = (m.frontmatter and m.frontmatter.date or ""):match("^(%d%d%d%d%-%d%d%-%d%d)")
        if date then
          if not by_date[date] then
            by_date[date] = {}
            table.insert(dates, date)
          end
          table.insert(by_date[date], m)
        end
      end
      table.sort(dates)
      for _, date in ipairs(dates) do
        local day_meetings = by_date[date]
        table.sort(day_meetings, function(a, b)
          local ha = a.frontmatter and a.frontmatter.hour or ""
          local hb = b.frontmatter and b.frontmatter.hour or ""
          return ha < hb
        end)
        -- Compute day of week name from date string
        local y, mo, d = date:match("(%d+)-(%d+)-(%d+)")
        local ts = os.time({ year = tonumber(y), month = tonumber(mo), day = tonumber(d) })
        local wday = os.date("*t", ts).wday -- 1=Sunday
        local day_name = day_names[wday == 1 and 7 or (wday - 1)]
        table.insert(lines, "")
        table.insert(lines, "### " .. day_name .. " (" .. date .. ")")
        for _, m in ipairs(day_meetings) do
          local hour = m.frontmatter and m.frontmatter.hour or "??:??"
          table.insert(lines, "  - " .. hour .. " " .. m.title)
        end
      end
    else
      table.insert(lines, "  (none)")
    end

    -- Collect selectable items by line number
    local selectable = {}
    local all_notes = {}
    vim.list_extend(all_notes, meetings)
    vim.list_extend(all_notes, topics)
    for line_idx, l in ipairs(lines) do
      -- Match "  - HH:MM title" or "  - title"
      local title = l:match("^  %- %d%d:%d%d (.+)") or l:match("^  %- (.+)")
      if title then
        for _, n in ipairs(all_notes) do
          if n.title == title then
            selectable[line_idx] = n
            break
          end
        end
      end
    end

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
        get_notes_dir(function(dir)
          vim.cmd.edit(dir .. "/" .. note.path .. ".md")
        end)
      end
    end, opts)

    vim.keymap.set("n", "h", function()
      vim.api.nvim_win_close(win, true)
      M.weekly(data.prev_week)
    end, opts)

    vim.keymap.set("n", "l", function()
      vim.api.nvim_win_close(win, true)
      M.weekly(data.next_week)
    end, opts)

    vim.keymap.set("n", "e", function()
      vim.api.nvim_win_close(win, true)
      get_notes_dir(function(dir)
        vim.cmd.edit(dir .. "/" .. data.file_path)
      end)
    end, opts)
  end)
end

--- Get current buffer's relative note path (without .md), or nil if not in notes_dir.
---@param dir string
---@return string|nil
local function current_note_rel(dir)
  local buf_path = vim.api.nvim_buf_get_name(0)
  if buf_path == "" then
    return nil
  end
  local prefix = vim.pesc(dir) .. "/"
  if not buf_path:match("^" .. prefix) then
    return nil
  end
  return buf_path:gsub("^" .. prefix, ""):gsub("%.md$", "")
end

--- Create a new note with frontmatter, optionally pre-filling the path.
---@param prefill string|nil
local function create_note(prefill)
  get_notes_dir(function(dir)
    vim.ui.input({ prompt = "Note path (relative to notes): ", default = prefill or "" }, function(input)
      if not input or input == "" then
        return
      end

      local rel_path = input
      if not rel_path:match("%.md$") then
        rel_path = rel_path .. ".md"
      end

      local abs_path = dir .. "/" .. rel_path
      local title = vim.fn.fnamemodify(rel_path, ":t:r")

      local parent = vim.fn.fnamemodify(abs_path, ":h")
      vim.fn.mkdir(parent, "p")

      local content = "---\ntitle: " .. title .. "\ntags:\n---\n\n"
      local f = io.open(abs_path, "w")
      if f then
        f:write(content)
        f:close()
        vim.cmd.edit(abs_path)
        vim.api.nvim_win_set_cursor(0, { 6, 0 })
      else
        vim.notify("Failed to create: " .. abs_path, vim.log.levels.ERROR)
      end
    end)
  end)
end

--- Create a new note with frontmatter.
function M.new()
  create_note()
end

--- Create a new child note under the current note.
function M.new_child()
  get_notes_dir(function(dir)
    local rel = current_note_rel(dir)
    if not rel then
      vim.notify("Current buffer is not a note", vim.log.levels.WARN)
      return
    end
    create_note(rel .. "/")
  end)
end

--- Create a new sibling note beside the current note.
function M.new_sibling()
  get_notes_dir(function(dir)
    local rel = current_note_rel(dir)
    if not rel then
      vim.notify("Current buffer is not a note", vim.log.levels.WARN)
      return
    end
    local parent = rel:match("^(.+)/") or ""
    local prefill = parent ~= "" and (parent .. "/") or ""
    create_note(prefill)
  end)
end

--- Setup keybindings.
function M.setup(opts)
  opts = opts or {}
  if opts.binary then
    binary = opts.binary
  end
  if opts.notes_dir then
    notes_dir = vim.fn.expand(opts.notes_dir)
  end

  -- Global keybindings (available everywhere)
  vim.keymap.set("n", "<leader>vw", M.weekly, { desc = "Weekly view" })
  vim.keymap.set("n", "<leader>vo", M.open, { desc = "Open note" })
  vim.keymap.set("n", "<leader>vs", M.search, { desc = "Search notes" })
  local wk_ok, wk = pcall(require, "which-key")
  if wk_ok then
    wk.add({ "<leader>vn", group = "New note" })
  end
  vim.keymap.set("n", "<leader>vnn", M.new, { desc = "New note" })
  vim.keymap.set("n", "<leader>vnc", M.new_child, { desc = "New child note" })
  vim.keymap.set("n", "<leader>vns", M.new_sibling, { desc = "New sibling note" })
  vim.keymap.set("n", "<leader>vt", M.topics, { desc = "Topics" })
  vim.keymap.set("n", "<leader>vb", M.backlinks, { desc = "Backlinks" })

  -- Register blink.cmp wikilink completion source
  local has_blink, blink = pcall(require, "blink.cmp")
  if has_blink then
    blink.add_source_provider("vinote", {
      name = "vinote",
      module = "blink.cmp.sources.vinote",
      score_offset = 10,
    })
    blink.add_filetype_source("markdown", "vinote")
  end

  -- gf override only in markdown buffers
  local group = vim.api.nvim_create_augroup("vinote", { clear = true })
  vim.api.nvim_create_autocmd("FileType", {
    group = group,
    pattern = "markdown",
    callback = function(ev)
      vim.keymap.set("n", "gf", M.follow_link, { buffer = ev.buf, desc = "Follow wikilink" })
    end,
  })
end

return M
