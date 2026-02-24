export namespace main {
	
	export class Point {
	    checkers: number;
	    color: number;
	
	    static createFrom(source: any = {}) {
	        return new Point(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.checkers = source["checkers"];
	        this.color = source["color"];
	    }
	}
	export class Board {
	    points: Point[];
	    bearoff: number[];
	
	    static createFrom(source: any = {}) {
	        return new Board(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.points = this.convertValues(source["points"], Point);
	        this.bearoff = source["bearoff"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CheckerMove {
	    index: number;
	    analysisDepth: string;
	    analysisEngine?: string;
	    move: string;
	    equity: number;
	    equityError?: number;
	    playerWinChance: number;
	    playerGammonChance: number;
	    playerBackgammonChance: number;
	    opponentWinChance: number;
	    opponentGammonChance: number;
	    opponentBackgammonChance: number;
	
	    static createFrom(source: any = {}) {
	        return new CheckerMove(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.analysisDepth = source["analysisDepth"];
	        this.analysisEngine = source["analysisEngine"];
	        this.move = source["move"];
	        this.equity = source["equity"];
	        this.equityError = source["equityError"];
	        this.playerWinChance = source["playerWinChance"];
	        this.playerGammonChance = source["playerGammonChance"];
	        this.playerBackgammonChance = source["playerBackgammonChance"];
	        this.opponentWinChance = source["opponentWinChance"];
	        this.opponentGammonChance = source["opponentGammonChance"];
	        this.opponentBackgammonChance = source["opponentBackgammonChance"];
	    }
	}
	export class CheckerAnalysis {
	    moves: CheckerMove[];
	
	    static createFrom(source: any = {}) {
	        return new CheckerAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.moves = this.convertValues(source["moves"], CheckerMove);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Collection {
	    id: number;
	    name: string;
	    description: string;
	    sortOrder: number;
	    createdAt: string;
	    updatedAt: string;
	    positionCount: number;
	
	    static createFrom(source: any = {}) {
	        return new Collection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.sortOrder = source["sortOrder"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.positionCount = source["positionCount"];
	    }
	}
	export class Config {
	    window_width: number;
	    window_height: number;
	    last_database_path: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.window_width = source["window_width"];
	        this.window_height = source["window_height"];
	        this.last_database_path = source["last_database_path"];
	    }
	}
	export class Cube {
	    owner: number;
	    value: number;
	
	    static createFrom(source: any = {}) {
	        return new Cube(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.owner = source["owner"];
	        this.value = source["value"];
	    }
	}
	export class DoublingCubeAnalysis {
	    analysisDepth: string;
	    analysisEngine?: string;
	    playerWinChances: number;
	    playerGammonChances: number;
	    playerBackgammonChances: number;
	    opponentWinChances: number;
	    opponentGammonChances: number;
	    opponentBackgammonChances: number;
	    cubelessNoDoubleEquity: number;
	    cubelessDoubleEquity: number;
	    cubefulNoDoubleEquity: number;
	    cubefulNoDoubleError: number;
	    cubefulDoubleTakeEquity: number;
	    cubefulDoubleTakeError: number;
	    cubefulDoublePassEquity: number;
	    cubefulDoublePassError: number;
	    bestCubeAction: string;
	    wrongPassPercentage: number;
	    wrongTakePercentage: number;
	
	    static createFrom(source: any = {}) {
	        return new DoublingCubeAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.analysisDepth = source["analysisDepth"];
	        this.analysisEngine = source["analysisEngine"];
	        this.playerWinChances = source["playerWinChances"];
	        this.playerGammonChances = source["playerGammonChances"];
	        this.playerBackgammonChances = source["playerBackgammonChances"];
	        this.opponentWinChances = source["opponentWinChances"];
	        this.opponentGammonChances = source["opponentGammonChances"];
	        this.opponentBackgammonChances = source["opponentBackgammonChances"];
	        this.cubelessNoDoubleEquity = source["cubelessNoDoubleEquity"];
	        this.cubelessDoubleEquity = source["cubelessDoubleEquity"];
	        this.cubefulNoDoubleEquity = source["cubefulNoDoubleEquity"];
	        this.cubefulNoDoubleError = source["cubefulNoDoubleError"];
	        this.cubefulDoubleTakeEquity = source["cubefulDoubleTakeEquity"];
	        this.cubefulDoubleTakeError = source["cubefulDoubleTakeError"];
	        this.cubefulDoublePassEquity = source["cubefulDoublePassEquity"];
	        this.cubefulDoublePassError = source["cubefulDoublePassError"];
	        this.bestCubeAction = source["bestCubeAction"];
	        this.wrongPassPercentage = source["wrongPassPercentage"];
	        this.wrongTakePercentage = source["wrongTakePercentage"];
	    }
	}
	export class FileDialogResponse {
	    file_path: string;
	    content: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new FileDialogResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.content = source["content"];
	        this.error = source["error"];
	    }
	}
	export class Game {
	    id: number;
	    match_id: number;
	    game_number: number;
	    initial_score: number[];
	    winner: number;
	    points_won: number;
	    move_count: number;
	
	    static createFrom(source: any = {}) {
	        return new Game(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.match_id = source["match_id"];
	        this.game_number = source["game_number"];
	        this.initial_score = source["initial_score"];
	        this.winner = source["winner"];
	        this.points_won = source["points_won"];
	        this.move_count = source["move_count"];
	    }
	}
	export class Match {
	    id: number;
	    player1_name: string;
	    player2_name: string;
	    event: string;
	    location: string;
	    round: string;
	    match_length: number;
	    // Go type: time
	    match_date: any;
	    // Go type: time
	    import_date: any;
	    file_path: string;
	    game_count: number;
	    tournament_id?: number;
	    tournament_name: string;
	    last_visited_position: number;
	
	    static createFrom(source: any = {}) {
	        return new Match(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.player1_name = source["player1_name"];
	        this.player2_name = source["player2_name"];
	        this.event = source["event"];
	        this.location = source["location"];
	        this.round = source["round"];
	        this.match_length = source["match_length"];
	        this.match_date = this.convertValues(source["match_date"], null);
	        this.import_date = this.convertValues(source["import_date"], null);
	        this.file_path = source["file_path"];
	        this.game_count = source["game_count"];
	        this.tournament_id = source["tournament_id"];
	        this.tournament_name = source["tournament_name"];
	        this.last_visited_position = source["last_visited_position"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Position {
	    id: number;
	    board: Board;
	    cube: Cube;
	    dice: number[];
	    score: number[];
	    player_on_roll: number;
	    decision_type: number;
	    has_jacoby: number;
	    has_beaver: number;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.board = this.convertValues(source["board"], Board);
	        this.cube = this.convertValues(source["cube"], Cube);
	        this.dice = source["dice"];
	        this.score = source["score"];
	        this.player_on_roll = source["player_on_roll"];
	        this.decision_type = source["decision_type"];
	        this.has_jacoby = source["has_jacoby"];
	        this.has_beaver = source["has_beaver"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MatchMovePosition {
	    position: Position;
	    move_id: number;
	    game_id: number;
	    game_number: number;
	    move_number: number;
	    move_type: string;
	    player_on_roll: number;
	    player1_name: string;
	    player2_name: string;
	    checker_move: string;
	    cube_action: string;
	
	    static createFrom(source: any = {}) {
	        return new MatchMovePosition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.position = this.convertValues(source["position"], Position);
	        this.move_id = source["move_id"];
	        this.game_id = source["game_id"];
	        this.game_number = source["game_number"];
	        this.move_number = source["move_number"];
	        this.move_type = source["move_type"];
	        this.player_on_roll = source["player_on_roll"];
	        this.player1_name = source["player1_name"];
	        this.player2_name = source["player2_name"];
	        this.checker_move = source["checker_move"];
	        this.cube_action = source["cube_action"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Move {
	    id: number;
	    game_id: number;
	    move_number: number;
	    move_type: string;
	    position_id: number;
	    player: number;
	    dice: number[];
	    checker_move?: string;
	    cube_action?: string;
	
	    static createFrom(source: any = {}) {
	        return new Move(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.game_id = source["game_id"];
	        this.move_number = source["move_number"];
	        this.move_type = source["move_type"];
	        this.position_id = source["position_id"];
	        this.player = source["player"];
	        this.dice = source["dice"];
	        this.checker_move = source["checker_move"];
	        this.cube_action = source["cube_action"];
	    }
	}
	
	
	export class PositionAnalysis {
	    positionId: number;
	    xgid: string;
	    player1: string;
	    player2: string;
	    analysisType: string;
	    analysisEngineVersion: string;
	    doublingCubeAnalysis?: DoublingCubeAnalysis;
	    allCubeAnalyses?: DoublingCubeAnalysis[];
	    checkerAnalysis?: CheckerAnalysis;
	    playedMove?: string;
	    playedCubeAction?: string;
	    playedMoves?: string[];
	    playedCubeActions?: string[];
	    // Go type: time
	    creationDate: any;
	    // Go type: time
	    lastModifiedDate: any;
	
	    static createFrom(source: any = {}) {
	        return new PositionAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.positionId = source["positionId"];
	        this.xgid = source["xgid"];
	        this.player1 = source["player1"];
	        this.player2 = source["player2"];
	        this.analysisType = source["analysisType"];
	        this.analysisEngineVersion = source["analysisEngineVersion"];
	        this.doublingCubeAnalysis = this.convertValues(source["doublingCubeAnalysis"], DoublingCubeAnalysis);
	        this.allCubeAnalyses = this.convertValues(source["allCubeAnalyses"], DoublingCubeAnalysis);
	        this.checkerAnalysis = this.convertValues(source["checkerAnalysis"], CheckerAnalysis);
	        this.playedMove = source["playedMove"];
	        this.playedCubeAction = source["playedCubeAction"];
	        this.playedMoves = source["playedMoves"];
	        this.playedCubeActions = source["playedCubeActions"];
	        this.creationDate = this.convertValues(source["creationDate"], null);
	        this.lastModifiedDate = this.convertValues(source["lastModifiedDate"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchHistory {
	    id: number;
	    command: string;
	    position: string;
	    timestamp: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchHistory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.command = source["command"];
	        this.position = source["position"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class SessionState {
	    lastSearchCommand: string;
	    lastSearchPosition: string;
	    lastPositionIndex: number;
	    lastPositionIds: number[];
	    hasActiveSearch: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SessionState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lastSearchCommand = source["lastSearchCommand"];
	        this.lastSearchPosition = source["lastSearchPosition"];
	        this.lastPositionIndex = source["lastPositionIndex"];
	        this.lastPositionIds = source["lastPositionIds"];
	        this.hasActiveSearch = source["hasActiveSearch"];
	    }
	}
	export class Tournament {
	    id: number;
	    name: string;
	    date: string;
	    location: string;
	    sortOrder: number;
	    createdAt: string;
	    updatedAt: string;
	    matchCount: number;
	
	    static createFrom(source: any = {}) {
	        return new Tournament(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.date = source["date"];
	        this.location = source["location"];
	        this.sortOrder = source["sortOrder"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.matchCount = source["matchCount"];
	    }
	}

}

