import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import axios from 'axios';
import { Building2, ShieldCheck, CheckCircle2, ArrowLeft } from 'lucide-react';

const API_URL = import.meta.env.VITE_API_URL || 'https://guarderiabiometricback.onrender.com';

const CAMPO_VACIO = {
  nombre_guarderia: '',
  direccion: '',
  nombre_contacto: '',
  email_contacto: '',
  telefono_contacto: '',
  username_deseado: '',
  password: '',
};

// RegistroGuarderia es el formulario público de "quiero dar de alta mi
// guardería" -- NO crea la guardería ni la cuenta admin en el momento (ver
// solicitudes.go en el backend): solo manda una solicitud que queda
// pendiente hasta que el dueño de la plataforma la aprueba desde /plataforma.
const RegistroGuarderia = () => {
  const [form, setForm] = useState(CAMPO_VACIO);
  const [enviando, setEnviando] = useState(false);
  const [error, setError] = useState('');
  const [enviado, setEnviado] = useState(false);

  const actualizarCampo = (campo) => (e) => setForm({ ...form, [campo]: e.target.value });

  const enviar = async (e) => {
    e.preventDefault();
    setError('');
    if (form.password.length < 8) {
      setError('La contraseña debe tener al menos 8 caracteres.');
      return;
    }
    setEnviando(true);
    try {
      await axios.post(`${API_URL}/solicitudes-guarderia`, form);
      setEnviado(true);
    } catch (err) {
      setError(err.response?.data?.error || 'No se pudo enviar la solicitud. Intenta de nuevo.');
    } finally {
      setEnviando(false);
    }
  };

  if (enviado) {
    return (
      <div className="min-h-screen bg-paper flex items-center justify-center p-4">
        <div className="bg-white border border-slate-200 p-8 rounded-[2.5rem] w-full max-w-md shadow-xl text-center space-y-4">
          <div className="inline-flex bg-emerald-100 p-4 rounded-3xl text-emerald-600"><CheckCircle2 size={40} /></div>
          <h1 className="text-2xl font-black text-slate-900">Solicitud enviada</h1>
          <p className="text-slate-500 text-sm leading-relaxed">
            Recibimos los datos de <strong>{form.nombre_guarderia}</strong>. En cuanto se revise y apruebe tu
            solicitud, podrás iniciar sesión con el usuario que registraste aquí.
          </p>
          <Link to="/" className="inline-flex items-center gap-2 text-brand-600 font-bold text-sm mt-2">
            <ArrowLeft size={16} /> Volver al inicio
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-paper flex items-center justify-center p-4 py-10">
      <div className="bg-white border border-slate-200 p-8 rounded-[2.5rem] w-full max-w-lg shadow-xl">
        <div className="text-center mb-8">
          <div className="inline-flex bg-brand-600 p-4 rounded-3xl shadow-lg mb-4"><Building2 size={36} className="text-white" /></div>
          <h1 className="text-2xl font-black text-slate-900">Da de alta tu guardería</h1>
          <p className="text-slate-500 text-sm mt-1">Un formulario, revisamos tu solicitud y te avisamos para que empieces a usar Pasitos.</p>
        </div>

        <form onSubmit={enviar} className="space-y-4">
          <div>
            <label className="text-[10px] font-black uppercase text-slate-400 ml-4 tracking-widest">Nombre de la guardería</label>
            <input required value={form.nombre_guarderia} onChange={actualizarCampo('nombre_guarderia')}
              className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-slate-900 outline-none focus:ring-2 focus:ring-brand-500 transition-all mt-1"
              placeholder="Guardería Sol y Luna" />
          </div>
          <div>
            <label className="text-[10px] font-black uppercase text-slate-400 ml-4 tracking-widest">Dirección (opcional)</label>
            <input value={form.direccion} onChange={actualizarCampo('direccion')}
              className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-slate-900 outline-none focus:ring-2 focus:ring-brand-500 transition-all mt-1"
              placeholder="Calle, número, ciudad" />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[10px] font-black uppercase text-slate-400 ml-4 tracking-widest">Tu nombre</label>
              <input required value={form.nombre_contacto} onChange={actualizarCampo('nombre_contacto')}
                className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-slate-900 outline-none focus:ring-2 focus:ring-brand-500 transition-all mt-1"
                placeholder="Quién solicita" />
            </div>
            <div>
              <label className="text-[10px] font-black uppercase text-slate-400 ml-4 tracking-widest">Teléfono (opcional)</label>
              <input value={form.telefono_contacto} onChange={actualizarCampo('telefono_contacto')}
                className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-slate-900 outline-none focus:ring-2 focus:ring-brand-500 transition-all mt-1"
                placeholder="10 dígitos" />
            </div>
          </div>
          <div>
            <label className="text-[10px] font-black uppercase text-slate-400 ml-4 tracking-widest">Correo de contacto</label>
            <input required type="email" value={form.email_contacto} onChange={actualizarCampo('email_contacto')}
              className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-slate-900 outline-none focus:ring-2 focus:ring-brand-500 transition-all mt-1"
              placeholder="tu@correo.com" />
          </div>

          <div className="pt-2 border-t border-slate-100">
            <p className="text-[10px] font-black uppercase text-slate-400 tracking-widest mt-4 mb-3 flex items-center gap-2">
              <ShieldCheck size={14} className="text-brand-500" /> Tu cuenta de administrador (para cuando se apruebe)
            </p>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-[10px] font-black uppercase text-slate-400 ml-4 tracking-widest">Usuario</label>
                <input required minLength={3} value={form.username_deseado} onChange={actualizarCampo('username_deseado')}
                  className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-slate-900 outline-none focus:ring-2 focus:ring-brand-500 transition-all mt-1"
                  placeholder="admin_solyluna" />
              </div>
              <div>
                <label className="text-[10px] font-black uppercase text-slate-400 ml-4 tracking-widest">Contraseña</label>
                <input required type="password" minLength={8} value={form.password} onChange={actualizarCampo('password')}
                  className="w-full bg-slate-50 border border-slate-200 p-4 rounded-2xl text-slate-900 outline-none focus:ring-2 focus:ring-brand-500 transition-all mt-1"
                  placeholder="Mínimo 8 caracteres" />
              </div>
            </div>
          </div>

          {error && <p className="text-rose-600 font-bold bg-rose-50 p-4 rounded-xl text-center text-sm">{error}</p>}

          <p className="text-center text-[11px] text-slate-400 font-medium">
            Al enviar esta solicitud aceptas los{' '}
            <Link to="/terminos" target="_blank" rel="noreferrer" className="text-brand-600 font-bold hover:underline">
              Términos y Condiciones
            </Link>{' '}
            de Pasitos.
          </p>

          <button type="submit" disabled={enviando}
            className="w-full bg-brand-600 hover:bg-brand-700 text-white font-black py-4 rounded-2xl uppercase tracking-tight shadow-lg transition-all active:scale-95 disabled:opacity-50">
            {enviando ? 'Enviando...' : 'Enviar solicitud'}
          </button>
        </form>

        <Link to="/" className="flex items-center justify-center gap-2 text-slate-400 hover:text-slate-600 font-bold text-xs mt-6">
          <ArrowLeft size={14} /> Volver al inicio
        </Link>
      </div>
    </div>
  );
};

export default RegistroGuarderia;
