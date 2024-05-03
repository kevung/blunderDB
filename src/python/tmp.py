import os
import re
import sqlite3


###############################################
# PARSE DATA

def extract_metadata(text):
    metadata = {}
    metadata_pattern = re.compile(r'; \[([\w\s]+)\s"([\w\s\d\.:]+)"\]')

    matches = metadata_pattern.findall(text)

    for match in matches:
        metadata[match[0]] = match[1]

    return metadata

def extract_match_points(text):
    match_points = None
    metadata_pattern = re.compile(r'(\d+) point match', re.IGNORECASE)

    match = metadata_pattern.search(text)

    if match:
        match_points = int(match.group(1))

    return match_points

def parse_game_data(text):
    games_data = []
    game_pattern = re.compile(r'Game\s+(\d+)\s+Jab34\s+:\s+(\d+)\s+postmanpat\s+:\s+(\d+)(.*?)Game\s+\d+', re.DOTALL)
    moves_pattern = re.compile(r'(\d+)\)\s+(.*?)\s+(\d+): (.*?)\s+(\d+): (.*?)\n', re.DOTALL)

    games = game_pattern.findall(text)

    for game in games:
        game_number = game[0]
        player1_score = game[1]
        player2_score = game[2]
        moves_data = game[3]

        moves = moves_pattern.findall(moves_data)
        game_moves = [(move[1].strip(), move[3].strip(), move[5].strip()) for move in moves]
        games_data.append({
            "game_number": game_number,
            "player1_score": player1_score,
            "player2_score": player2_score,
            "moves": game_moves
        })

    return games_data

def open_game_file(fn):
    with open(fn, "r") as file:
        data = file.read()
    return data


###############################################
# DATABASE OPERATIONS

def connect_db(fn, overwrite):
    if overwrite:
        if os.path.exists(fn):
            os.remove(fn)
            print(f"The file '{fn}' has been deleted.")
    conn = sqlite3.connect(fn)
    return conn

def close_db(conn):
    conn.close()
    return True





###############################################


