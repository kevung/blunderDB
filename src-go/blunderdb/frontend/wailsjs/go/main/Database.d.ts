// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {main} from '../models';

export function DeleteAnalysis(arg1:number):Promise<void>;

export function DeleteComment(arg1:number):Promise<void>;

export function DeletePosition(arg1:number):Promise<void>;

export function LoadAllPositions():Promise<Array<main.Position>>;

export function LoadAnalysis(arg1:number):Promise<main.PositionAnalysis>;

export function LoadComment(arg1:number):Promise<string>;

export function LoadPosition(arg1:number):Promise<main.Position>;

export function LoadPositionsByCheckerPosition(arg1:main.Position,arg2:boolean,arg3:boolean,arg4:string,arg5:string,arg6:string,arg7:string,arg8:string,arg9:string,arg10:string,arg11:string,arg12:string,arg13:string,arg14:string,arg15:string,arg16:string,arg17:string):Promise<Array<main.Position>>;

export function PositionExists(arg1:main.Position):Promise<{[key: string]: any}>;

export function SaveAnalysis(arg1:number,arg2:main.PositionAnalysis):Promise<void>;

export function SaveComment(arg1:number,arg2:string):Promise<void>;

export function SavePosition(arg1:main.Position):Promise<number>;

export function SetupDatabase(arg1:string):Promise<void>;

export function UpdatePosition(arg1:main.Position):Promise<void>;
