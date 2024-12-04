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
	    board: Board;
	    cube: Cube;
	    dice: number[];
	    score: number[];
	    player_on_roll: number;
	    decision_type: number;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.board = this.convertValues(source["board"], Board);
	        this.cube = this.convertValues(source["cube"], Cube);
	        this.dice = source["dice"];
	        this.score = source["score"];
	        this.player_on_roll = source["player_on_roll"];
	        this.decision_type = source["decision_type"];
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

