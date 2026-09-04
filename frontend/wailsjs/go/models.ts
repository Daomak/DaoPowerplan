export namespace main {
	
	export class PossibleValue {
	    index: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new PossibleValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.name = source["name"];
	    }
	}
	export class PowerPlan {
	    guid: string;
	    name: string;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PowerPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.guid = source["guid"];
	        this.name = source["name"];
	        this.active = source["active"];
	    }
	}
	export class PowerSetting {
	    groupGuid: string;
	    groupGuid2: string;
	    settingGuid: string;
	    settingGuid2: string;
	    name: string;
	    currentAC: string;
	    currentDC: string;
	    possibleValues: PossibleValue[];
	
	    static createFrom(source: any = {}) {
	        return new PowerSetting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.groupGuid = source["groupGuid"];
	        this.groupGuid2 = source["groupGuid2"];
	        this.settingGuid = source["settingGuid"];
	        this.settingGuid2 = source["settingGuid2"];
	        this.name = source["name"];
	        this.currentAC = source["currentAC"];
	        this.currentDC = source["currentDC"];
	        this.possibleValues = this.convertValues(source["possibleValues"], PossibleValue);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
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

