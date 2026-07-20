import React, { useState, useEffect } from 'react';
import api from './axiosConfig';
import { Save, X, Image as ImageIcon, Moon, Utensils, Info, Plus, Trash2, CheckCircle2 } from 'lucide-react';

const ComidaSelector = ({ label, icon: Icon, value, name, onChange }) => {
  const opciones = [
    { label: 'NADA', color: 'peer-checked:bg-rose-500 peer-checked:text-white', bg: 'bg-rose-50 text-rose-600' },
    { label: 'POCO', color: 'peer-checked:bg-orange-500 peer-checked:text-white', bg: 'bg-orange-50 text-orange-600' },
    { label: 'BIEN', color: 'peer-checked:bg-emerald-500 peer-checked:text-white', bg: 'bg-emerald-50 text-emerald-600' },
  ];

  return (
    <div className="flex flex-col gap-3 bg-white p-4 rounded-[2rem] border border-slate-100 shadow-sm">
      <label className="flex items-center gap-2 text-[10px] font-black uppercase text-slate-400 tracking-widest ml-1">
        <Icon size={14} className="text-violet-500" />
        {label}
      </label>
      <div className="flex gap-2">
        {opciones.map((opt) => (
          <label key={opt.label} className="flex-1 cursor-pointer">
            <input 
              type="radio" 
              name={name} 
              value={opt.label} 
              checked={value === opt.label}
              onChange={onChange} 
              className="sr-only peer" 
            />
            <div className={`text-center py-2 rounded-xl text-[10px] font-black transition-all border border-transparent peer-checked:shadow-lg active:scale-95 ${opt.bg} ${opt.color}`}>
              {opt.label}
            </div>
          </label>
        ))}
      </div>
    </div>
  );
};

const FormularioBitacora = ({ niñoId, nombreNiño, onCerrar }) => {
  const [formData, setFormData] = useState({
    desayuno: '', comida: '', merienda: '',
    esfinter: '', observaciones: '', durmio: false
  });
  
  const [padreInfo, setPadreInfo] = useState({ id: null, telefono: '' });
  const [fotosExistentes, setFotosExistentes] = useState([]); 
  const [nuevasFotos, setNuevasFotos] = useState([]);       
  const [previews, setPreviews] = useState([]);             
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const cargarDatos = async () => {
      if (!niñoId) return;
      try {
        setLoading(true);
        const res = await api.get(`/seguimiento/${niñoId}`);
        console.log("🔍 Datos iniciales cargados:", res.data);
        if (res.data) {
          const datosBitacora = res.data.bitacora || res.data;
          setFormData({
            desayuno: datosBitacora.desayuno || '',
            comida: datosBitacora.comida || '',
            merienda: datosBitacora.merienda || '',
            esfinter: datosBitacora.esfinter || '',
            observaciones: datosBitacora.observaciones || '',
            durmio: datosBitacora.durmio || false
          });
          setFotosExistentes(res.data.urls || []);
          setPadreInfo({
            id: res.data.padre_id,
            telefono: res.data.telefono_padre 
          });
        }
      } catch (err) {
        console.warn("⚠️ No hay bitácora previa para hoy");
      } finally { setLoading(false); }
    };
    cargarDatos();
  }, [niñoId]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleEsfinterChange = (tipo) => {
    let actual = formData.esfinter.split(', ').filter(Boolean);
    if (actual.includes(tipo)) {
      actual = actual.filter(t => t !== tipo);
    } else {
      actual.push(tipo);
    }
    setFormData(prev => ({ ...prev, esfinter: actual.join(', ') }));
  };

  const handleFileChange = (e) => {
    const files = Array.from(e.target.files);
    setNuevasFotos(prev => [...prev, ...files]);
    const nuevasPreviews = files.map(file => URL.createObjectURL(file));
    setPreviews(prev => [...prev, ...nuevasPreviews]);
  };

  const eliminarFotoNueva = (index) => {
    setNuevasFotos(prev => prev.filter((_, i) => i !== index));
    setPreviews(prev => prev.filter((_, i) => i !== index));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    console.log("🚀 Iniciando proceso de guardado...");
    
    const data = new FormData();
    Object.keys(formData).forEach(key => data.append(key, formData[key]));
    data.append("hijo_id", niñoId);
    nuevasFotos.forEach(foto => data.append("fotos", foto));

    try {
      setLoading(true);
      const response = await api.post('/seguimiento', data);
      
      // --- BLOQUE DE LOGS DE DIAGNÓSTICO ---
      console.log("✅ Servidor respondió con éxito");
      console.log("📦 Cuerpo de la respuesta (Raw):", response.data);

      const idToken = response.data.padre_id;
      const telf = response.data.telefono_padre;

      console.log("📱 Datos extraídos para WhatsApp:");
      console.log("   - ID del Padre (Token):", idToken);
      console.log("   - Teléfono del Padre:", telf);

      if (idToken && telf) {
        const urlPublica = `${window.location.origin}/seguimiento/${idToken}`;
        const mensajeWA = `¡Hola! He actualizado la bitácora de seguimiento de ${nombreNiño}. Puedes ver el reporte y las fotos aquí: ${urlPublica}`;
        const linkWA = `https://wa.me/${telf}?text=${encodeURIComponent(mensajeWA)}`;
        
        console.log("🔗 LINK FINAL GENERADO:", linkWA);
        console.log("🛰️ Intentando abrir WhatsApp en nueva pestaña...");
        
        const win = window.open(linkWA, '_blank');
        if (win) {
            console.log("🟢 Ventana abierta satisfactoriamente.");
        } else {
            console.error("🔴 ERROR: El navegador bloqueó la ventana emergente (Pop-up).");
            alert("Por favor habilita las ventanas emergentes para enviar el WhatsApp.");
        }
      } else {
        console.warn("⚠️ No se pudo generar el envío de WhatsApp:");
        if (!idToken) console.warn("   - 'padre_id' no llegó en la respuesta.");
        if (!telf) console.warn("   - 'telefono_padre' no llegó en la respuesta.");
      }
      // --------------------------------------
      
      onCerrar();
    } catch (err) { 
      console.error("❌ Error en la petición POST:", err);
      alert("Error al guardar la bitácora"); 
    } finally { setLoading(false); }
  };

  if (loading) return <div className="text-center p-20 font-black text-slate-400 animate-pulse uppercase tracking-[0.2em] text-xs">Procesando...</div>;

  return (
    <div className="bg-white rounded-[3rem] shadow-2xl border border-slate-100 overflow-hidden animate-in zoom-in duration-300 max-w-4xl w-full mx-auto">
      <div className="bg-violet-600 p-8 text-white flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-black uppercase tracking-tighter leading-none">Seguimiento Diario</h2>
          <p className="text-violet-200 text-xs font-bold uppercase tracking-widest mt-2">{nombreNiño}</p>
        </div>
        <button onClick={onCerrar} className="p-3 bg-white/10 hover:bg-white/20 rounded-2xl transition-all">
          <X size={24} />
        </button>
      </div>

      <form onSubmit={handleSubmit} className="p-8 space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="flex items-center justify-between bg-slate-50 p-5 rounded-[2rem] border border-slate-100">
            <div className="flex items-center gap-4">
              <div className={`p-3 rounded-xl ${formData.durmio ? 'bg-violet-600 text-white shadow-lg shadow-violet-200' : 'bg-slate-200 text-slate-400'}`}>
                <Moon size={20} />
              </div>
              <span className="font-black text-slate-900 uppercase text-xs tracking-tight">¿Durmió?</span>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" className="sr-only peer" checked={formData.durmio} onChange={(e) => setFormData({...formData, durmio: e.target.checked})} />
              <div className="w-12 h-7 bg-slate-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[4px] after:left-[4px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-violet-600"></div>
            </label>
          </div>

          <div className="flex flex-col gap-3 bg-slate-50 p-5 rounded-[2rem] border border-slate-100">
             <label className="flex items-center gap-2 text-[10px] font-black uppercase text-slate-400 tracking-widest">
               <CheckCircle2 size={14} className="text-violet-500" /> Esfínter
             </label>
             <div className="flex gap-2">
                {['PIPI', 'POPO'].map(tipo => (
                  <button
                    key={tipo}
                    type="button"
                    onClick={() => handleEsfinterChange(tipo)}
                    className={`flex-1 py-2 rounded-xl text-[10px] font-black transition-all border ${formData.esfinter.includes(tipo) ? 'bg-violet-600 text-white border-violet-600 shadow-md' : 'bg-white text-slate-400 border-slate-200'}`}
                  >
                    {tipo}
                  </button>
                ))}
             </div>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <ComidaSelector label="Desayuno" icon={Utensils} name="desayuno" value={formData.desayuno} onChange={handleChange} />
          <ComidaSelector label="Comida" icon={Utensils} name="comida" value={formData.comida} onChange={handleChange} />
          <ComidaSelector label="Merienda" icon={Utensils} name="merienda" value={formData.merienda} onChange={handleChange} />
        </div>

        <div className="flex flex-col gap-2">
          <label className="flex items-center gap-2 text-[10px] font-black uppercase text-slate-400 tracking-widest ml-2">
            <Info size={14} className="text-violet-500" /> Observaciones Generales
          </label>
          <textarea
            className="w-full bg-slate-50 border border-slate-100 rounded-[2.5rem] p-6 text-sm font-medium outline-none focus:ring-2 focus:ring-violet-500 transition-all min-h-[120px] resize-none shadow-inner"
            placeholder="Escribe aquí cualquier detalle adicional del día..."
            value={formData.observaciones}
            name="observaciones"
            onChange={handleChange}
          />
        </div>

        <div className="space-y-4">
          <label className="flex items-center gap-2 text-[10px] font-black uppercase text-slate-400 tracking-widest ml-2">
            <ImageIcon size={14} className="text-violet-500" /> Evidencia Fotográfica
          </label>
          <div className="grid grid-cols-2 sm:grid-cols-5 gap-4">
            <label className="aspect-square bg-slate-50 border-2 border-dashed border-slate-200 rounded-[2rem] flex flex-col items-center justify-center gap-2 cursor-pointer hover:bg-slate-100 hover:border-violet-300 transition-all group">
              <Plus size={24} className="text-slate-400 group-hover:text-violet-600" />
              <input type="file" multiple accept="image/*" className="hidden" onChange={handleFileChange} />
            </label>
            {previews.map((url, index) => (
              <div key={index} className="aspect-square rounded-[2rem] overflow-hidden relative group shadow-md">
                <img src={url} alt="preview" className="w-full h-full object-cover" />
                <button type="button" onClick={() => eliminarFotoNueva(index)} className="absolute top-2 right-2 p-2 bg-red-500 text-white rounded-xl opacity-0 group-hover:opacity-100 transition-opacity">
                  <Trash2 size={14} />
                </button>
              </div>
            ))}
            {fotosExistentes.map((url, index) => (
              <div key={`exp-${index}`} className="aspect-square rounded-[2rem] overflow-hidden relative border-2 border-violet-100">
                <img src={url} alt="existente" className="w-full h-full object-cover opacity-80" />
              </div>
            ))}
          </div>
        </div>

        <div className="flex gap-4 pt-4">
          <button type="submit" className="flex-1 bg-violet-600 hover:bg-violet-700 text-white font-black uppercase tracking-widest py-5 rounded-[2rem] shadow-xl shadow-violet-200 flex items-center justify-center gap-3 transition-all active:scale-95">
            <Save size={20} /> Guardar y Notificar
          </button>
          <button type="button" onClick={onCerrar} className="px-8 bg-slate-100 hover:bg-slate-200 text-slate-500 font-black uppercase tracking-widest rounded-[2rem] transition-all">
            Cerrar
          </button>
        </div>
      </form>
    </div>
  );
};

export default FormularioBitacora;