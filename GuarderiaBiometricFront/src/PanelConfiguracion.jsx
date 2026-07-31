import React, { useState, useEffect } from 'react';
import api from './axiosConfig';
import { ShieldCheck, Save, Loader2, Users, FileText } from 'lucide-react';
import { mostrarExito, mostrarError } from './utils/alertas';

const PanelConfiguracion = () => {
  const [texto, setTexto] = useState('');
  const [version, setVersion] = useState('');
  const [configurado, setConfigurado] = useState(false);
  const [totalConsentimientos, setTotalConsentimientos] = useState(null);
  const [loading, setLoading] = useState(true);
  const [guardando, setGuardando] = useState(false);

  const cargar = async () => {
    setLoading(true);
    try {
      const [resAviso, resEstadisticas] = await Promise.all([
        api.get('/aviso-privacidad'),
        api.get('/admin/aviso-privacidad/estadisticas'),
      ]);
      setTexto(resAviso.data.texto || '');
      setVersion(resAviso.data.version || '');
      setConfigurado(!!resAviso.data.configurado);
      setTotalConsentimientos(resEstadisticas.data.total_consentimientos ?? 0);
    } catch (err) {
      console.error('Error al cargar la configuración de privacidad:', err);
      mostrarError('No se pudo cargar el Aviso de Privacidad');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { cargar(); }, []);

  const guardar = async () => {
    if (!texto.trim()) {
      mostrarError('El texto del Aviso de Privacidad no puede quedar vacío');
      return;
    }
    setGuardando(true);
    try {
      const res = await api.put('/admin/aviso-privacidad', { texto: texto.trim() });
      mostrarExito(`Se guardó una nueva versión (${res.data.version}). Los tutores que ya habían aceptado una versión anterior no necesitan volver a hacerlo, pero cualquier registro nuevo verá este texto.`);
      cargar();
    } catch (err) {
      console.error('Error al guardar el Aviso de Privacidad:', err);
      mostrarError('No se pudo guardar el Aviso de Privacidad');
    } finally {
      setGuardando(false);
    }
  };

  return (
    <div className="animate-in fade-in duration-500">
      <div className="bg-white p-6 sm:p-8 rounded-[2.5rem] border border-slate-200 shadow-xl">
        <div className="flex items-center gap-4 mb-6">
          <div className="bg-brand-100 p-3 rounded-2xl text-brand-600"><ShieldCheck size={28} /></div>
          <div>
            <h3 className="text-xl font-black uppercase text-slate-900">Aviso de Privacidad</h3>
            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Cumplimiento LFPDPPP</p>
          </div>
        </div>

        {loading ? (
          <div className="py-20 text-center text-slate-400 font-black uppercase tracking-widest text-xs">Cargando...</div>
        ) : (
          <div className="space-y-6">
            {!configurado && (
              <div className="bg-amber-50 border border-amber-200 rounded-2xl p-5 text-amber-800 text-sm font-bold">
                Mientras no guardes un texto aquí, el kiosco no permitirá registrar nuevos tutores.
                Pide a tu asesor legal el texto del Aviso de Privacidad para datos biométricos y de
                menores, y pégalo abajo.
              </div>
            )}

            <div className="flex flex-wrap gap-4">
              <div className="bg-slate-50 border border-slate-100 rounded-2xl px-5 py-4 flex items-center gap-3">
                <FileText size={20} className="text-brand-500" />
                <div>
                  <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Versión vigente</p>
                  <p className="font-black text-slate-900">{version || 'Sin configurar'}</p>
                </div>
              </div>
              <div className="bg-slate-50 border border-slate-100 rounded-2xl px-5 py-4 flex items-center gap-3">
                <Users size={20} className="text-brand-500" />
                <div>
                  <p className="text-[9px] font-black text-slate-400 uppercase tracking-widest">Consentimientos registrados</p>
                  <p className="font-black text-slate-900">{totalConsentimientos}</p>
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-[10px] font-black uppercase text-slate-400 ml-2 tracking-widest">
                Texto del Aviso de Privacidad
              </label>
              <textarea
                rows={14}
                value={texto}
                onChange={(e) => setTexto(e.target.value)}
                placeholder="Pega aquí el texto completo del Aviso de Privacidad..."
                className="w-full bg-slate-50 border border-slate-200 p-5 rounded-2xl outline-none focus:ring-2 focus:ring-brand-500 text-slate-900 font-medium resize-y"
              />
              <p className="text-[10px] text-slate-400 ml-2">
                Al guardar se crea una nueva versión automáticamente. Los tutores lo verán completo,
                con scroll, antes de registrar su rostro en el kiosco.
              </p>
            </div>

            <button
              onClick={guardar}
              disabled={guardando}
              className="w-full sm:w-auto flex items-center justify-center gap-3 bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white font-black uppercase px-8 py-4 rounded-2xl shadow-lg transition-all active:scale-95"
            >
              {guardando ? <Loader2 className="animate-spin" size={20} /> : <Save size={20} />}
              {guardando ? 'Guardando...' : 'Guardar Aviso de Privacidad'}
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default PanelConfiguracion;
