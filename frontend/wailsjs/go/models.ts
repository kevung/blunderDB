export namespace database {
	
	export class BlunderEntry {
	    PositionID: number;
	    MatchID: number;
	    TournamentID: number;
	    ErrorMP: number;
	    MWCLoss: number;
	    Description: string;
	    DecisionType: number;
	    MatchDate: string;
	    PlayerNames: string;
	
	    static createFrom(source: any = {}) {
	        return new BlunderEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PositionID = source["PositionID"];
	        this.MatchID = source["MatchID"];
	        this.TournamentID = source["TournamentID"];
	        this.ErrorMP = source["ErrorMP"];
	        this.MWCLoss = source["MWCLoss"];
	        this.Description = source["Description"];
	        this.DecisionType = source["DecisionType"];
	        this.MatchDate = source["MatchDate"];
	        this.PlayerNames = source["PlayerNames"];
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
	export class CubeActionStats {
	    Action: string;
	    PR: number;
	    MWC: number;
	    NumDecisions: number;
	    BlunderCount: number;
	
	    static createFrom(source: any = {}) {
	        return new CubeActionStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Action = source["Action"];
	        this.PR = source["PR"];
	        this.MWC = source["MWC"];
	        this.NumDecisions = source["NumDecisions"];
	        this.BlunderCount = source["BlunderCount"];
	    }
	}
	export class ErrorBucket {
	    MinMP: number;
	    MaxMP: number;
	    Count: number;
	
	    static createFrom(source: any = {}) {
	        return new ErrorBucket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.MinMP = source["MinMP"];
	        this.MaxMP = source["MaxMP"];
	        this.Count = source["Count"];
	    }
	}
	export class MatchPlayerDetailStats {
	    total_decisions: number;
	    total_errors: number;
	    total_blunders: number;
	    total_equity_error: number;
	    pr: number;
	    mwc_loss: number;
	    checker_decisions: number;
	    checker_errors: number;
	    checker_blunders: number;
	    checker_equity_error: number;
	    pr_checker: number;
	    checker_mwc_loss: number;
	    double_decisions: number;
	    double_errors: number;
	    double_blunders: number;
	    double_equity_error: number;
	    double_mwc_loss: number;
	    take_decisions: number;
	    take_errors: number;
	    take_blunders: number;
	    take_equity_error: number;
	    take_mwc_loss: number;
	    pr_cube: number;
	    cube_mwc_loss: number;
	    snowie_er: number;
	
	    static createFrom(source: any = {}) {
	        return new MatchPlayerDetailStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_decisions = source["total_decisions"];
	        this.total_errors = source["total_errors"];
	        this.total_blunders = source["total_blunders"];
	        this.total_equity_error = source["total_equity_error"];
	        this.pr = source["pr"];
	        this.mwc_loss = source["mwc_loss"];
	        this.checker_decisions = source["checker_decisions"];
	        this.checker_errors = source["checker_errors"];
	        this.checker_blunders = source["checker_blunders"];
	        this.checker_equity_error = source["checker_equity_error"];
	        this.pr_checker = source["pr_checker"];
	        this.checker_mwc_loss = source["checker_mwc_loss"];
	        this.double_decisions = source["double_decisions"];
	        this.double_errors = source["double_errors"];
	        this.double_blunders = source["double_blunders"];
	        this.double_equity_error = source["double_equity_error"];
	        this.double_mwc_loss = source["double_mwc_loss"];
	        this.take_decisions = source["take_decisions"];
	        this.take_errors = source["take_errors"];
	        this.take_blunders = source["take_blunders"];
	        this.take_equity_error = source["take_equity_error"];
	        this.take_mwc_loss = source["take_mwc_loss"];
	        this.pr_cube = source["pr_cube"];
	        this.cube_mwc_loss = source["cube_mwc_loss"];
	        this.snowie_er = source["snowie_er"];
	    }
	}
	export class MatchDetailStats {
	    match_id: number;
	    player1: MatchPlayerDetailStats;
	    player2: MatchPlayerDetailStats;
	
	    static createFrom(source: any = {}) {
	        return new MatchDetailStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.match_id = source["match_id"];
	        this.player1 = this.convertValues(source["player1"], MatchPlayerDetailStats);
	        this.player2 = this.convertValues(source["player2"], MatchPlayerDetailStats);
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
	
	export class MatchStats {
	    ID: number;
	    Date: string;
	    PlayerName: string;
	    PR: number;
	    MWC: number;
	    NumDecisions: number;
	
	    static createFrom(source: any = {}) {
	        return new MatchStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Date = source["Date"];
	        this.PlayerName = source["PlayerName"];
	        this.PR = source["PR"];
	        this.MWC = source["MWC"];
	        this.NumDecisions = source["NumDecisions"];
	    }
	}
	export class PlayerFrequency {
	    Name: string;
	    Count: number;
	
	    static createFrom(source: any = {}) {
	        return new PlayerFrequency(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Count = source["Count"];
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
	export class SelectionSpec {
	    Kind: string;
	    CubeAction: string;
	    BucketMinMP: number;
	    BucketMaxMP: number;
	    TournamentID: number;
	    MatchID: number;
	    LastN: number;
	    PositionID: number;
	    OnlyWithError: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SelectionSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Kind = source["Kind"];
	        this.CubeAction = source["CubeAction"];
	        this.BucketMinMP = source["BucketMinMP"];
	        this.BucketMaxMP = source["BucketMaxMP"];
	        this.TournamentID = source["TournamentID"];
	        this.MatchID = source["MatchID"];
	        this.LastN = source["LastN"];
	        this.PositionID = source["PositionID"];
	        this.OnlyWithError = source["OnlyWithError"];
	    }
	}
	export class SessionState {
	    lastSearchCommand: string;
	    lastSearchPosition: string;
	    lastPositionIndex: number;
	    lastPositionIds: number[];
	    hasActiveSearch: boolean;
	    viewsJSON: string;
	
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
	        this.viewsJSON = source["viewsJSON"];
	    }
	}
	export class StatsDateRange {
	    DateFrom: string;
	    DateTo: string;
	
	    static createFrom(source: any = {}) {
	        return new StatsDateRange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DateFrom = source["DateFrom"];
	        this.DateTo = source["DateTo"];
	    }
	}
	export class StatsFilter {
	    PlayerName: string;
	    TournamentIDs: number[];
	    DateFrom: string;
	    DateTo: string;
	    DecisionType: number;
	    MatchLength: number[];
	
	    static createFrom(source: any = {}) {
	        return new StatsFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PlayerName = source["PlayerName"];
	        this.TournamentIDs = source["TournamentIDs"];
	        this.DateFrom = source["DateFrom"];
	        this.DateTo = source["DateTo"];
	        this.DecisionType = source["DecisionType"];
	        this.MatchLength = source["MatchLength"];
	    }
	}
	export class TournamentStats {
	    ID: number;
	    Name: string;
	    Date: string;
	    PR: number;
	    MWC: number;
	    NumDecisions: number;
	
	    static createFrom(source: any = {}) {
	        return new TournamentStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Name = source["Name"];
	        this.Date = source["Date"];
	        this.PR = source["PR"];
	        this.MWC = source["MWC"];
	        this.NumDecisions = source["NumDecisions"];
	    }
	}
	export class StatsTotals {
	    NumPositions: number;
	    NumMatches: number;
	    NumTournaments: number;
	    NumDecisions: number;
	
	    static createFrom(source: any = {}) {
	        return new StatsTotals(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.NumPositions = source["NumPositions"];
	        this.NumMatches = source["NumMatches"];
	        this.NumTournaments = source["NumTournaments"];
	        this.NumDecisions = source["NumDecisions"];
	    }
	}
	export class StatsResult {
	    Totals: StatsTotals;
	    PRGlobal: number;
	    PRChecker: number;
	    PRCube: number;
	    PRRolling: Record<number, number>;
	    MWCGlobal: number;
	    MWCChecker: number;
	    MWCCube: number;
	    MWCRolling: Record<number, number>;
	    MWCAvailable: boolean;
	    SnowieGlobal: number;
	    PerTournament: TournamentStats[];
	    PerMatch: MatchStats[];
	    CubeActionBreakdown: CubeActionStats[];
	    ErrorHistogram: ErrorBucket[];
	    TopBlunders: BlunderEntry[];
	
	    static createFrom(source: any = {}) {
	        return new StatsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Totals = this.convertValues(source["Totals"], StatsTotals);
	        this.PRGlobal = source["PRGlobal"];
	        this.PRChecker = source["PRChecker"];
	        this.PRCube = source["PRCube"];
	        this.PRRolling = source["PRRolling"];
	        this.MWCGlobal = source["MWCGlobal"];
	        this.MWCChecker = source["MWCChecker"];
	        this.MWCCube = source["MWCCube"];
	        this.MWCRolling = source["MWCRolling"];
	        this.MWCAvailable = source["MWCAvailable"];
	        this.SnowieGlobal = source["SnowieGlobal"];
	        this.PerTournament = this.convertValues(source["PerTournament"], TournamentStats);
	        this.PerMatch = this.convertValues(source["PerMatch"], MatchStats);
	        this.CubeActionBreakdown = this.convertValues(source["CubeActionBreakdown"], CubeActionStats);
	        this.ErrorHistogram = this.convertValues(source["ErrorHistogram"], ErrorBucket);
	        this.TopBlunders = this.convertValues(source["TopBlunders"], BlunderEntry);
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

export namespace domain {
	
	export class AnkiCard {
	    id: number;
	    deckId: number;
	    positionId: number;
	    due: string;
	    stability: number;
	    difficulty: number;
	    elapsedDays: number;
	    scheduledDays: number;
	    reps: number;
	    lapses: number;
	    state: number;
	    lastReview: string;
	
	    static createFrom(source: any = {}) {
	        return new AnkiCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.deckId = source["deckId"];
	        this.positionId = source["positionId"];
	        this.due = source["due"];
	        this.stability = source["stability"];
	        this.difficulty = source["difficulty"];
	        this.elapsedDays = source["elapsedDays"];
	        this.scheduledDays = source["scheduledDays"];
	        this.reps = source["reps"];
	        this.lapses = source["lapses"];
	        this.state = source["state"];
	        this.lastReview = source["lastReview"];
	    }
	}
	export class AnkiDeck {
	    id: number;
	    name: string;
	    description: string;
	    sourceType: string;
	    sourceId: number;
	    sourceCommand: string;
	    requestRetention: number;
	    maximumInterval: number;
	    enableFuzz: boolean;
	    cardCount: number;
	    dueCount: number;
	    newCount: number;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AnkiDeck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.sourceType = source["sourceType"];
	        this.sourceId = source["sourceId"];
	        this.sourceCommand = source["sourceCommand"];
	        this.requestRetention = source["requestRetention"];
	        this.maximumInterval = source["maximumInterval"];
	        this.enableFuzz = source["enableFuzz"];
	        this.cardCount = source["cardCount"];
	        this.dueCount = source["dueCount"];
	        this.newCount = source["newCount"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class AnkiDeckStats {
	    newCount: number;
	    learningCount: number;
	    reviewCount: number;
	    dueCount: number;
	    totalCount: number;
	
	    static createFrom(source: any = {}) {
	        return new AnkiDeckStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.newCount = source["newCount"];
	        this.learningCount = source["learningCount"];
	        this.reviewCount = source["reviewCount"];
	        this.dueCount = source["dueCount"];
	        this.totalCount = source["totalCount"];
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
	export class AnkiReviewCard {
	    card: AnkiCard;
	    position: Position;
	
	    static createFrom(source: any = {}) {
	        return new AnkiReviewCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.card = this.convertValues(source["card"], AnkiCard);
	        this.position = this.convertValues(source["position"], Position);
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
	
	export class CommentEntry {
	    id: number;
	    positionId: number;
	    text: string;
	    createdAt: string;
	    modifiedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CommentEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.positionId = source["positionId"];
	        this.text = source["text"];
	        this.createdAt = source["createdAt"];
	        this.modifiedAt = source["modifiedAt"];
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
	export class ExportOptions {
	    exportPath: string;
	    positions: Position[];
	    metadata: Record<string, string>;
	    includeAnalysis: boolean;
	    includeComments: boolean;
	    includeFilterLibrary: boolean;
	    includePlayedMoves: boolean;
	    includeMatches: boolean;
	    includeCollections: boolean;
	    collectionIDs: number[];
	    matchIDs: number[];
	    tournamentIDs: number[];
	
	    static createFrom(source: any = {}) {
	        return new ExportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exportPath = source["exportPath"];
	        this.positions = this.convertValues(source["positions"], Position);
	        this.metadata = source["metadata"];
	        this.includeAnalysis = source["includeAnalysis"];
	        this.includeComments = source["includeComments"];
	        this.includeFilterLibrary = source["includeFilterLibrary"];
	        this.includePlayedMoves = source["includePlayedMoves"];
	        this.includeMatches = source["includeMatches"];
	        this.includeCollections = source["includeCollections"];
	        this.collectionIDs = source["collectionIDs"];
	        this.matchIDs = source["matchIDs"];
	        this.tournamentIDs = source["tournamentIDs"];
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
	    comment: string;
	    tournament_sort_order: number;
	    pr: number;
	    mwc_loss: number;
	    pr2: number;
	    mwc_loss2: number;
	
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
	        this.comment = source["comment"];
	        this.tournament_sort_order = source["tournament_sort_order"];
	        this.pr = source["pr"];
	        this.mwc_loss = source["mwc_loss"];
	        this.pr2 = source["pr2"];
	        this.mwc_loss2 = source["mwc_loss2"];
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
	export class SearchFilters {
	    filter: Position;
	    includeCube: boolean;
	    includeScore: boolean;
	    pipCountFilter: string;
	    winRateFilter: string;
	    gammonRateFilter: string;
	    backgammonRateFilter: string;
	    player2WinRateFilter: string;
	    player2GammonRateFilter: string;
	    player2BackgammonRateFilter: string;
	    player1CheckerOffFilter: string;
	    player2CheckerOffFilter: string;
	    player1BackCheckerFilter: string;
	    player2BackCheckerFilter: string;
	    player1CheckerInZoneFilter: string;
	    player2CheckerInZoneFilter: string;
	    searchText: string;
	    player1AbsolutePipCountFilter: string;
	    equityFilter: string;
	    decisionTypeFilter: boolean;
	    diceRollFilter: boolean;
	    movePatternFilter: string;
	    dateFilter: string;
	    player1OutfieldBlotFilter: string;
	    player2OutfieldBlotFilter: string;
	    player1JanBlotFilter: string;
	    player2JanBlotFilter: string;
	    noContactFilter: boolean;
	    mirrorFilter: boolean;
	    moveErrorFilter: string;
	    matchIDsFilter: string;
	    tournamentIDsFilter: string;
	    restrictToPositionIDs: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchFilters(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filter = this.convertValues(source["filter"], Position);
	        this.includeCube = source["includeCube"];
	        this.includeScore = source["includeScore"];
	        this.pipCountFilter = source["pipCountFilter"];
	        this.winRateFilter = source["winRateFilter"];
	        this.gammonRateFilter = source["gammonRateFilter"];
	        this.backgammonRateFilter = source["backgammonRateFilter"];
	        this.player2WinRateFilter = source["player2WinRateFilter"];
	        this.player2GammonRateFilter = source["player2GammonRateFilter"];
	        this.player2BackgammonRateFilter = source["player2BackgammonRateFilter"];
	        this.player1CheckerOffFilter = source["player1CheckerOffFilter"];
	        this.player2CheckerOffFilter = source["player2CheckerOffFilter"];
	        this.player1BackCheckerFilter = source["player1BackCheckerFilter"];
	        this.player2BackCheckerFilter = source["player2BackCheckerFilter"];
	        this.player1CheckerInZoneFilter = source["player1CheckerInZoneFilter"];
	        this.player2CheckerInZoneFilter = source["player2CheckerInZoneFilter"];
	        this.searchText = source["searchText"];
	        this.player1AbsolutePipCountFilter = source["player1AbsolutePipCountFilter"];
	        this.equityFilter = source["equityFilter"];
	        this.decisionTypeFilter = source["decisionTypeFilter"];
	        this.diceRollFilter = source["diceRollFilter"];
	        this.movePatternFilter = source["movePatternFilter"];
	        this.dateFilter = source["dateFilter"];
	        this.player1OutfieldBlotFilter = source["player1OutfieldBlotFilter"];
	        this.player2OutfieldBlotFilter = source["player2OutfieldBlotFilter"];
	        this.player1JanBlotFilter = source["player1JanBlotFilter"];
	        this.player2JanBlotFilter = source["player2JanBlotFilter"];
	        this.noContactFilter = source["noContactFilter"];
	        this.mirrorFilter = source["mirrorFilter"];
	        this.moveErrorFilter = source["moveErrorFilter"];
	        this.matchIDsFilter = source["matchIDsFilter"];
	        this.tournamentIDsFilter = source["tournamentIDsFilter"];
	        this.restrictToPositionIDs = source["restrictToPositionIDs"];
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
	export class Tournament {
	    id: number;
	    name: string;
	    date: string;
	    location: string;
	    sortOrder: number;
	    createdAt: string;
	    updatedAt: string;
	    matchCount: number;
	    comment: string;
	    pr: number;
	    mwc_loss: number;
	
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
	        this.comment = source["comment"];
	        this.pr = source["pr"];
	        this.mwc_loss = source["mwc_loss"];
	    }
	}

}

export namespace main {
	
	export class StatsFilterPersisted {
	    player_name: string;
	    tournament_ids: number[];
	    date_from: string;
	    date_to: string;
	    decision_type?: number;
	    match_length: number[];
	    metric: string;
	
	    static createFrom(source: any = {}) {
	        return new StatsFilterPersisted(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.player_name = source["player_name"];
	        this.tournament_ids = source["tournament_ids"];
	        this.date_from = source["date_from"];
	        this.date_to = source["date_to"];
	        this.decision_type = source["decision_type"];
	        this.match_length = source["match_length"];
	        this.metric = source["metric"];
	    }
	}
	export class Config {
	    window_width: number;
	    window_height: number;
	    last_database_path: string;
	    stats_filter?: StatsFilterPersisted;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.window_width = source["window_width"];
	        this.window_height = source["window_height"];
	        this.last_database_path = source["last_database_path"];
	        this.stats_filter = this.convertValues(source["stats_filter"], StatsFilterPersisted);
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

}

export namespace sql {
	
	export class DB {
	
	
	    static createFrom(source: any = {}) {
	        return new DB(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

