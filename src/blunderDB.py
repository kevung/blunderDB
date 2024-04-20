import re


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

def main():
    with open("./share/match_example1.txt", "r") as file:
        data = file.read()

    games_data = parse_game_data(data)

    for game_data in games_data:
        print(f"Game {game_data['game_number']}:")
        print(f"Jab34: {game_data['player1_score']}  postmanpat: {game_data['player2_score']}")
        for move_number, player1_move, player2_move in game_data['moves']:
            print(f"  {move_number}: Jab34 {player1_move}  postmanpat {player2_move}")

if __name__ == "__main__":
    main()
