import sys, gi
gi.require_version('Gtk','3.0'); gi.require_version('WebKit2','4.1')
from gi.repository import Gtk, WebKit2, GLib
url, cross = sys.argv[1], sys.argv[2] == 'cross'
attr = 'crossorigin="anonymous"' if cross else ''
html = f'''<!doctype html><body><video id=v {attr} muted preload=auto src="{url}"></video><script>
const v=document.getElementById('v'); const r=[];
function done(s){{document.title='DONE '+s}}
v.onerror=()=>done('videoerror '+(v.error&&v.error.code));
v.addEventListener('loadeddata',()=>{{v.currentTime=12.5}},{{once:true}});
v.addEventListener('seeked',()=>{{
 try{{const c=document.createElement('canvas');c.width=v.videoWidth;c.height=v.videoHeight;
 const g=c.getContext('2d');g.drawImage(v,0,0);const d=g.getImageData(c.width>>1,c.height>>1,1,1).data;
 r.push('getImageData OK px='+Array.from(d).join(','));}}catch(e){{r.push('ERR '+e.name)}}
 r.push('t='+v.currentTime.toFixed(2)+' size='+v.videoWidth+'x'+v.videoHeight+' origin='+location.origin);
 v.play().then(()=>setTimeout(()=>{{r.push('play t='+v.currentTime.toFixed(2));done(r.join(' | '))}},1500)).catch(e=>done(r.join(' | ')+' | playERR '+e.name));
}},{{once:true}});
setTimeout(()=>done('timeout '+r.join(' | ')),20000);
</script>'''
ctx = WebKit2.WebContext.get_default()
def handler(req):
    b = GLib.Bytes.new(html.encode())
    from gi.repository import Gio
    req.finish(Gio.MemoryInputStream.new_from_bytes(b), len(html.encode()), 'text/html')
ctx.register_uri_scheme('wails', handler)
w = Gtk.Window(); wv = WebKit2.WebView.new_with_context(ctx); w.add(wv); w.show_all()
def on_title(*a):
    t = wv.get_title() or ''
    if t.startswith('DONE'):
        print(t[5:], flush=True); Gtk.main_quit()
wv.connect('notify::title', on_title)
wv.load_uri('wails://wails/')
GLib.timeout_add_seconds(30, lambda: (print('hard timeout'), Gtk.main_quit()))
Gtk.main()
