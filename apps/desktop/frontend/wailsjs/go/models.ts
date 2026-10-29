export namespace main {
	
	export class SubscriptionBatchItem {
	    subscription_id: number;
	    kind: string;
	    payload?: any;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionBatchItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subscription_id = source["subscription_id"];
	        this.kind = source["kind"];
	        this.payload = source["payload"];
	    }
	}
	export class VariableWriteAttempt {
	    sent: boolean;
	    expected_revision?: string;
	    response_json?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new VariableWriteAttempt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sent = source["sent"];
	        this.expected_revision = source["expected_revision"];
	        this.response_json = source["response_json"];
	        this.error = source["error"];
	    }
	}

}

