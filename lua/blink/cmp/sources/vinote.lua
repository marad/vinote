--- blink.cmp source for wikilink completion in vinote.
--- Triggers after [[ and completes note paths with closing ]].

local vinote = {}

function vinote.new(opts, config)
  local self = setmetatable({}, { __index = vinote })
  self.cache = nil
  return self
end

function vinote:get_trigger_characters()
  return { "[" }
end

--- Check if cursor is inside [[ and return the byte offset (1-indexed) of the
--- first character after "[[", or nil if the cursor is not inside a wikilink.
--- @return number|nil
local function get_wikilink_start(context)
  local col = context.cursor[2] -- 0-indexed byte position of cursor
  local before = context.line:sub(1, col)

  local start = nil
  local i = 1
  while true do
    local s = before:find("%[%[", i, false)
    if not s then break end
    local close = before:find("%]%]", s + 2, true)
    if not close then
      start = s + 2 -- 1-indexed byte after [[
    end
    i = s + 1
  end
  return start
end

function vinote:get_completions(context, callback)
  local start = get_wikilink_start(context)
  if not start then
    callback()
    return
  end

  local cursor_line = context.cursor[1] - 1 -- 0-indexed for LSP
  local cursor_col = context.cursor[2] -- already 0-indexed byte offset

  if self.cache then
    callback({
      is_incomplete_forward = false,
      is_incomplete_backward = false,
      items = self:_make_items(cursor_line, start - 1, cursor_col),
    })
    return
  end

  local stdout = {}
  vim.fn.jobstart({ "vn", "query", "--all", "--json" }, {
    stdout_buffered = true,
    on_stdout = function(_, data)
      stdout = data
    end,
    on_exit = function(_, code)
      if code ~= 0 then
        vim.schedule(function() callback() end)
        return
      end
      vim.schedule(function()
        local output = table.concat(stdout, "\n")
        local ok, notes = pcall(vim.json.decode, output)
        if not ok or type(notes) ~= "table" then
          callback()
          return
        end
        self.cache = notes
        callback({
          is_incomplete_forward = false,
          is_incomplete_backward = false,
          items = self:_make_items(cursor_line, start - 1, cursor_col),
        })
      end)
    end,
  })

  return function() end
end

--- Build completion items with textEdit that replaces from after [[ to cursor
--- and appends ]].
--- @param line number 0-indexed line
--- @param start_char number 0-indexed character after [[
--- @param end_char number 0-indexed cursor character
function vinote:_make_items(line, start_char, end_char)
  local kind_ref = require("blink.cmp.types").CompletionItemKind.Reference
  local items = {}
  for _, note in ipairs(self.cache or {}) do
    items[#items + 1] = {
      label = note.path,
      kind = kind_ref,
      insertTextFormat = vim.lsp.protocol.InsertTextFormat.PlainText,
      textEdit = {
        newText = note.path,
        range = {
          start = { line = line, character = start_char },
          ["end"] = { line = line, character = end_char },
        },
      },
    }
  end
  return items
end

function vinote:should_show_items(context, items)
  return get_wikilink_start(context) ~= nil
end

function vinote:reload()
  self.cache = nil
end

return vinote
