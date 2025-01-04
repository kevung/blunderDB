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
	export class PositionAnalysis {
	    positionId: number;
	    xgid: string;
	    player1: string;
	    player2: string;
	    analysisType: string;
	    analysisEngineVersion: string;
	    doublingCubeAnalysis?: DoublingCubeAnalysis;
	    checkerAnalysis?: CheckerAnalysis;
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
	        this.checkerAnalysis = this.convertValues(source["checkerAnalysis"], CheckerAnalysis);
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

}

