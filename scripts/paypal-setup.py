#!/usr/bin/env python3
"""Creates Withdo Pro's PayPal product, its four plans and the webhook (docs/plans/m17-paypal.md).

    scripts/paypal-setup.py <credentials.json>

credentials.json is {"client_id", "secret", "env": "sandbox"|"live"}; keep it out of the repo. It
prints the ids (none is secret) and writes them to ids.json. PayPal-Request-Id makes a rerun
return the same objects instead of making new ones.
"""
import json,subprocess,sys
c=json.load(open(sys.argv[1]))
BASE='https://api-m.paypal.com' if c.get('env')=='live' else 'https://api-m.sandbox.paypal.com'
def curl(args,stdin=None):
    r=subprocess.run(['curl','-s']+args,input=stdin,capture_output=True,text=True); return json.loads(r.stdout or '{}')
tok=curl(['-K','-','-X','POST',BASE+'/v1/oauth2/token','-d','grant_type=client_credentials'],'user = "%s:%s"'%(c['client_id'],c['secret']))['access_token']
H=['-H','Content-Type: application/json','-H','Authorization: Bearer '+tok,'-H','Prefer: return=representation']
def post(path,body,key):
    return curl(['-X','POST',BASE+path]+H+['-H','PayPal-Request-Id: withdo-'+key,'-d',json.dumps(body)])
prod=post('/v1/catalogs/products',{'name':'Withdo Pro','description':'The official model in Withdo: 100 requests a day on every device.','type':'SERVICE','category':'SOFTWARE','home_url':'https://withdo.si/'},'product-1')
pid=prod['id']
def plan(name,unit,price,trial,key):
    cycles=[]
    seq=1
    if trial:
        cycles.append({'frequency':{'interval_unit':'DAY','interval_count':7},'tenure_type':'TRIAL','sequence':1,'total_cycles':1,'pricing_scheme':{'fixed_price':{'value':'0','currency_code':'USD'}}}); seq=2
    cycles.append({'frequency':{'interval_unit':unit,'interval_count':1},'tenure_type':'REGULAR','sequence':seq,'total_cycles':0,'pricing_scheme':{'fixed_price':{'value':price,'currency_code':'USD'}}})
    p=post('/v1/billing/plans',{'product_id':pid,'name':name,'status':'ACTIVE','billing_cycles':cycles,'payment_preferences':{'auto_bill_outstanding':True,'payment_failure_threshold':1,'setup_fee':{'value':'0','currency_code':'USD'},'setup_fee_failure_action':'CANCEL'}},key)
    return p.get('id') or p
ids={'product':pid,
 'monthly_trial':plan('Withdo Pro — Monthly (7-day trial)','MONTH','6.99',True,'plan-mt-1'),
 'monthly':plan('Withdo Pro — Monthly','MONTH','6.99',False,'plan-m-1'),
 'yearly_trial':plan('Withdo Pro — Yearly (7-day trial)','YEAR','58.72',True,'plan-yt-1'),
 'yearly':plan('Withdo Pro — Yearly','YEAR','58.72',False,'plan-y-1')}
events=['BILLING.SUBSCRIPTION.ACTIVATED','BILLING.SUBSCRIPTION.UPDATED','BILLING.SUBSCRIPTION.CANCELLED','BILLING.SUBSCRIPTION.SUSPENDED','BILLING.SUBSCRIPTION.EXPIRED','BILLING.SUBSCRIPTION.RE-ACTIVATED','BILLING.SUBSCRIPTION.PAYMENT.FAILED','PAYMENT.SALE.COMPLETED','PAYMENT.SALE.REFUNDED']
wh=post('/v1/notifications/webhooks',{'url':'https://api.withdo.si/v1/billing/paypal','event_types':[{'name':e} for e in events]},'webhook-1')
ids['webhook']=wh.get('id') or wh
json.dump(ids,open('ids.json','w'),indent=1)
print(json.dumps(ids,indent=1))
