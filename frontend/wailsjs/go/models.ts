export namespace app {
	
	export class MovedPath {
	    from: string;
	    to: string;
	
	    static createFrom(source: any = {}) {
	        return new MovedPath(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	    }
	}
	export class ChangeReport {
	    newCount: number;
	    modifiedCount: number;
	    missingCount: number;
	    movedCount: number;
	    new: string[];
	    modified: string[];
	    missing: string[];
	    moved: MovedPath[];
	
	    static createFrom(source: any = {}) {
	        return new ChangeReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.newCount = source["newCount"];
	        this.modifiedCount = source["modifiedCount"];
	        this.missingCount = source["missingCount"];
	        this.movedCount = source["movedCount"];
	        this.new = source["new"];
	        this.modified = source["modified"];
	        this.missing = source["missing"];
	        this.moved = this.convertValues(source["moved"], MovedPath);
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
	export class Detail {
	    file: store.File;
	    absPath: string;
	    tags: store.Tag[];
	    albums: store.Album[];
	    sidecars: string[];
	
	    static createFrom(source: any = {}) {
	        return new Detail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file = this.convertValues(source["file"], store.File);
	        this.absPath = source["absPath"];
	        this.tags = this.convertValues(source["tags"], store.Tag);
	        this.albums = this.convertValues(source["albums"], store.Album);
	        this.sidecars = source["sidecars"];
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
	
	export class OrganizePreview {
	    count: number;
	    noDate: number;
	    moves: organize.Move[];
	
	    static createFrom(source: any = {}) {
	        return new OrganizePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.noDate = source["noDate"];
	        this.moves = this.convertValues(source["moves"], organize.Move);
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
	export class Overview {
	    stats: store.Stats;
	    dirs: store.DirCount[];
	    years: store.YearCount[];
	    tags: store.Tag[];
	    albums: store.Album[];
	
	    static createFrom(source: any = {}) {
	        return new Overview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stats = this.convertValues(source["stats"], store.Stats);
	        this.dirs = this.convertValues(source["dirs"], store.DirCount);
	        this.years = this.convertValues(source["years"], store.YearCount);
	        this.tags = this.convertValues(source["tags"], store.Tag);
	        this.albums = this.convertValues(source["albums"], store.Album);
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
	export class State {
	    root: string;
	    recent: string[];
	    language: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.recent = source["recent"];
	        this.language = source["language"];
	        this.version = source["version"];
	    }
	}

}

export namespace library {
	
	export class DateResult {
	    written: number;
	    indexed: number;
	
	    static createFrom(source: any = {}) {
	        return new DateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.written = source["written"];
	        this.indexed = source["indexed"];
	    }
	}
	export class TakeoutResult {
	    trashed: number;
	    dates: number;
	    skipped: number;
	
	    static createFrom(source: any = {}) {
	        return new TakeoutResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trashed = source["trashed"];
	        this.dates = source["dates"];
	        this.skipped = source["skipped"];
	    }
	}
	export class TakeoutSummary {
	    sidecars: number;
	    dates: number;
	
	    static createFrom(source: any = {}) {
	        return new TakeoutSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sidecars = source["sidecars"];
	        this.dates = source["dates"];
	    }
	}

}

export namespace organize {
	
	export class Move {
	    id: number;
	    from: string;
	    to: string;
	
	    static createFrom(source: any = {}) {
	        return new Move(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.from = source["from"];
	        this.to = source["to"];
	    }
	}

}

export namespace store {
	
	export class Album {
	    id: number;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new Album(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class DirCount {
	    dir: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new DirCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dir = source["dir"];
	        this.count = source["count"];
	    }
	}
	export class File {
	    id: number;
	    relPath: string;
	    dir: string;
	    name: string;
	    ext: string;
	    kind: string;
	    size: number;
	    mtime: number;
	    hash: string;
	    takenAt: number;
	    takenSrc: string;
	    width: number;
	    height: number;
	    orientation: number;
	    camera: string;
	    hasGps: boolean;
	    lat: number;
	    lon: number;
	    metaJson: string;
	    sidecarOf: number;
	    status: string;
	    trashPath: string;
	    dateOverride: number;
	
	    static createFrom(source: any = {}) {
	        return new File(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.relPath = source["relPath"];
	        this.dir = source["dir"];
	        this.name = source["name"];
	        this.ext = source["ext"];
	        this.kind = source["kind"];
	        this.size = source["size"];
	        this.mtime = source["mtime"];
	        this.hash = source["hash"];
	        this.takenAt = source["takenAt"];
	        this.takenSrc = source["takenSrc"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.orientation = source["orientation"];
	        this.camera = source["camera"];
	        this.hasGps = source["hasGps"];
	        this.lat = source["lat"];
	        this.lon = source["lon"];
	        this.metaJson = source["metaJson"];
	        this.sidecarOf = source["sidecarOf"];
	        this.status = source["status"];
	        this.trashPath = source["trashPath"];
	        this.dateOverride = source["dateOverride"];
	    }
	}
	export class Filter {
	    inDir: boolean;
	    dir: string;
	    recursive: boolean;
	    kind: string;
	    tagId: number;
	    albumId: number;
	    search: string;
	    status: string;
	    noDate: boolean;
	    year: number;
	    sort: string;
	    offset: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new Filter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inDir = source["inDir"];
	        this.dir = source["dir"];
	        this.recursive = source["recursive"];
	        this.kind = source["kind"];
	        this.tagId = source["tagId"];
	        this.albumId = source["albumId"];
	        this.search = source["search"];
	        this.status = source["status"];
	        this.noDate = source["noDate"];
	        this.year = source["year"];
	        this.sort = source["sort"];
	        this.offset = source["offset"];
	        this.limit = source["limit"];
	    }
	}
	export class Item {
	    id: number;
	    name: string;
	    relPath: string;
	    kind: string;
	    size: number;
	    takenAt: number;
	    takenSrc: string;
	    hash: string;
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.relPath = source["relPath"];
	        this.kind = source["kind"];
	        this.size = source["size"];
	        this.takenAt = source["takenAt"];
	        this.takenSrc = source["takenSrc"];
	        this.hash = source["hash"];
	    }
	}
	export class Page {
	    items: Item[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new Page(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Item);
	        this.total = source["total"];
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
	export class Stats {
	    files: number;
	    media: number;
	    other: number;
	    noDate: number;
	    trashed: number;
	    missing: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = source["files"];
	        this.media = source["media"];
	        this.other = source["other"];
	        this.noDate = source["noDate"];
	        this.trashed = source["trashed"];
	        this.missing = source["missing"];
	    }
	}
	export class Tag {
	    id: number;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new Tag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class YearCount {
	    year: number;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new YearCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.count = source["count"];
	    }
	}

}

