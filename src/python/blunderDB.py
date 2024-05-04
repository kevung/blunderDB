import sqlite3

__NAME__ = "blunderDB"
__VERSION__ = "0.1.0"
__DBNAME__ = "blunder.db"

def add_column_if_not_exists(table_name, column_name, column_type):
    pass

def main():
    print("blunderDB, version ", __VERSION__)

    conn = sqlite3.connect(__DBNAME__)
    c = conn.cursor()

    c.execute("""CREATE TABLE IF NOT EXISTS Metadata (
        program_name TEXT(20),
        program_version TEXT(8),
        owner TEXT(20),
        last_modified TEXT(10)
        )""")

    c.execute("SELECT COUNT(*) FROM Metadata")
    if c.fetchone()[0] == 0:
        c.execute("""INSERT INTO Metadata ('program_name', 'program_version')
                  VALUES(\"%s\", \"%s\") """ % (__NAME__, __VERSION__) )

    c.execute("""CREATE TABLE IF NOT EXISTS Player (
        id INTEGER PRIMARY KEY,
        name TEXT(20),
        comment TEXT
        )""")

    c.execute("""CREATE TABLE IF NOT EXISTS Tag (
        id INTEGER PRIMARY KEY,
        name TEXT(20)
        )""")

    c.execute("""CREATE TABLE IF NOT EXISTS Position (
        id INTEGER PRIMARY KEY,
        p1 INTEGER,
        p2 INTEGER,
        p3 INTEGER,
        p4 INTEGER,
        p5 INTEGER,
        p6 INTEGER,
        p7 INTEGER,
        p8 INTEGER,
        p9 INTEGER,
        p10 INTEGER,
        p11 INTEGER,
        p12 INTEGER,
        p13 INTEGER,
        p14 INTEGER,
        p15 INTEGER,
        p16 INTEGER,
        p17 INTEGER,
        p18 INTEGER,
        p19 INTEGER,
        p20 INTEGER,
        p21 INTEGER,
        p22 INTEGER,
        p23 INTEGER,
        p24 INTEGER,
        bar_player1 INTEGER,
        bar_player2 INTEGER,
        die_1 INTEGER,
        die_2 INTEGER,
        cube INTEGER,
        score_player1 INTEGER,
        score_player2 INTEGER,
        crawford INTEGER,
        jacoby INTEGER,
        beaver INTEGER,
        player1_id INTEGER,
        player2_id INTEGER,
        player_on_roll INTEGER,
        comment TEXT(200),
        date TEXT,
        FOREIGN KEY (player1_id) REFERENCES Player(id),
        FOREIGN KEY (player2_id) REFERENCES Player(id)
        )""")

    c.execute("""CREATE TABLE IF NOT EXISTS PositionTagMapping (
    position_id INTEGER,
    tag_id INTEGER,
    FOREIGN KEY (position_id) REFERENCES Position(id),
    FOREIGN KEY (tag_id) REFERENCES Tag(id)
            )""")

    c.execute("""CREATE TABLE IF NOT EXISTS Move (
            )""")

    c.execute("""CREATE TABLE IF NOT EXISTS Game (
        id INTEGER PRIMARY KEY,
        move_player1 TEXT,
        move_player2 TEXT
        )""")

    c.execute("""CREATE TABLE IF NOT EXISTS GameList (
        id INTEGER PRIMARY KEY,
        game_id INTEGER,
        score_player1 INTEGER,
        score_player2 INTEGER,
        FOREIGN KEY (game_id) REFERENCES Game(id)
        )""")

    c.execute("""CREATE TABLE IF NOT EXISTS Match (
        id INTEGER PRIMARY KEY,
        player1_id INTEGER,
        player2_id INTEGER,
        match_length INTEGER,
        event_name TEXT(20),
        event_date TEXT,
        event_time TEXT,
        event_place TEXT,
        comment TEXT,
        gamelist_id INTEGER,
        FOREIGN KEY (player1_id) REFERENCES Player(id),
        FOREIGN KEY (player2_id) REFERENCES Player(id),
        FOREIGN KEY (gamelist_id) REFERENCES GameList(id)
        )""")

    conn.commit()
    conn.close()

if __name__ == "__main__":
    main()
