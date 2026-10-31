-- Banc V0.5 : annotation de t_lisible_ms dans annot.csv, coup par coup, dans mpv.
-- Usage : mpv --script=annoter.lua --script-opts=annoter-csv=annot.csv VIDEO
-- Entrée = dés lisibles ici · x = illisible · Retour arrière = coup précédent · r = revoir

local opts = { csv = "annot.csv", avant = 5, apres = 4, vitesse = 0.5 }
require("mp.options").read_options(opts, "annoter")

local done_path = opts.csv .. ".fait"
local header, rows = nil, {}
local fait = {}
local i = 1

local function split(line)
    local t = {}
    for f in (line .. ","):gmatch("([^,]*),") do t[#t + 1] = f end
    return t
end

local function load()
    for line in io.lines(opts.csv) do
        line = line:gsub("\r$", "")
        if not header then header = line
        elseif line ~= "" then rows[#rows + 1] = split(line) end
    end
    local f = io.open(done_path)
    if f then
        for n in f:lines() do fait[n] = true end
        f:close()
    end
end

local function save()
    local f = assert(io.open(opts.csv .. ".tmp", "w"))
    f:write(header, "\n")
    for _, r in ipairs(rows) do f:write(table.concat(r, ","), "\n") end
    f:close()
    os.rename(opts.csv .. ".tmp", opts.csv)
    local d = assert(io.open(done_path, "w"))
    for n in pairs(fait) do d:write(n, "\n") end
    d:close()
end

local function count_done()
    local c = 0
    for _ in pairs(fait) do c = c + 1 end
    return c
end

local function roll_s() return tonumber(rows[i][3]) / 1000 end

local function osd()
    local r = rows[i]
    local d = r[2]
    local etat = fait[r[1]] and ((r[5] ~= "" and "noté " .. r[5] .. " ms") or "illisible") or "à faire"
    mp.set_osd_ass(0, 0, string.format(
        "{\\an7\\fs22\\bord2}Coup %d/%d  —  dés attendus %s-%s  —  %s  (%d faits)\\N"
            .. "{\\fs16}Espace lecture/pause · , . image par image · Entrée = lisibles ici · x = illisible · "
            .. "r = revoir · Retour arrière = précédent",
        i, #rows, d:sub(1, 1), d:sub(2, 2), etat, count_done()))
end

local function go(k)
    if k > #rows then
        mp.set_osd_ass(0, 0, "{\\an5\\fs36\\bord2}Terminé : " .. count_done() .. " coups. q pour quitter.")
        mp.set_property_bool("pause", true)
        i = #rows + 1
        return
    end
    i = math.max(1, k)
    mp.commandv("seek", tostring(math.max(0, roll_s() - opts.avant)), "absolute+exact")
    mp.set_property_number("speed", opts.vitesse)
    mp.set_property_bool("pause", false)
    osd()
end

local function next_todo(from)
    local k = from
    while k <= #rows and fait[rows[k][1]] do k = k + 1 end
    return k
end

local function mark(value)
    if i > #rows then return end
    rows[i][5] = value
    fait[rows[i][1]] = true
    save()
    go(next_todo(i + 1))
end

mp.observe_property("time-pos", "number", function(_, t)
    if t and i <= #rows and not mp.get_property_bool("pause") and t > roll_s() + opts.apres then
        mp.set_property_bool("pause", true)
        mp.osd_message("Fin de l'extrait : r pour revoir, x si illisible", 3)
    end
end)

mp.add_forced_key_binding("ENTER", "annoter-lisible", function()
    local t = mp.get_property_number("time-pos")
    if t then mark(tostring(math.floor(t * 1000 + 0.5))) end
end)
mp.add_forced_key_binding("x", "annoter-illisible", function() mark("") end)
mp.add_forced_key_binding("r", "annoter-revoir", function() go(i) end)
mp.add_forced_key_binding("BS", "annoter-precedent", function() go(i - 1) end)

local started = false
mp.register_event("file-loaded", function()
    if started then return end
    started = true
    load()
    go(next_todo(1))
end)
