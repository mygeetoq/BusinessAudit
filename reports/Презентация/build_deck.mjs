import fs from 'node:fs/promises';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { Presentation, PresentationFile } from '@oai/artifact-tool';

const workspaceDir='/Users/avtoshin/university/managing_projects';
const skillDir='/Users/avtoshin/.codex/plugins/cache/openai-primary-runtime/presentations/26.909.12148/skills/presentations';
const outDir=path.join(workspaceDir,'reports/Презентация');
const assets=path.join(workspaceDir,'reports/assets');
const finalPath=path.join(outDir,'Бизнес_Пульс_проект.pptx');
const stagingDir=path.join(workspaceDir,'reports/.pptx-build');
await fs.mkdir(outDir,{recursive:true}); await fs.mkdir(stagingDir,{recursive:true});
const { finalizePresentation } = await import(pathToFileURL(path.join(skillDir,'container_tools/artifact_tool_utils.mjs')).href);
const ppt=Presentation.create({slideSize:{width:1280,height:720}});
const C={navy:'#0E1B32', blue:'#1E63B7', light:'#F4F7FB', white:'#FFFFFF', muted:'#5F6B7D', green:'#2E8874', orange:'#D98A32', border:'#DDE5EE'};
const font='Arial';
const title='Разработка системы мониторинга и прогнозирования денежных потоков малого бизнеса';
function shape(s,x,y,w,h,fill,r=0){return s.shapes.add({geometry:r?'roundRect':'rect',position:{left:x,top:y,width:w,height:h},fill,line:{fill:'none',width:0},borderRadius:r||undefined});}
function txt(s,t,x,y,w,h,size=24,color=C.navy,bold=false){const q=s.shapes.add({geometry:'textbox',position:{left:x,top:y,width:w,height:h},fill:'none',line:{fill:'none',width:0}});q.text=t;q.text.style={typeface:font,fontSize:size,bold,color,autoFit:'shrinkText',verticalAlignment:'middle'};return q;}
function base(n,head,sub=''){const s=ppt.slides.add();s.background.fill=C.light;shape(s,0,0,1280,12,C.blue);txt(s,head,64,40,1140,72,38,C.navy,true);if(sub)txt(s,sub,66,109,1120,44,20,C.muted);shape(s,64,667,1152,1,C.border);txt(s,`Бизнес.Пульс  ·  учебный проект  ·  сентябрь 2026`,64,675,900,25,14,C.muted);txt(s,String(n).padStart(2,'0'),1164,673,50,28,16,C.blue,true);return s;}
async function photo(s,name,x,y,w,h){s.images.add({blob:new Uint8Array(await fs.readFile(path.join(assets,name))),contentType:'image/png',alt:name,fit:'contain',position:{left:x,top:y,width:w,height:h}});}
function card(s,x,y,w,h,heading,body,color=C.blue){shape(s,x,y,w,h,C.white,18);shape(s,x,y,7,h,color);txt(s,heading,x+25,y+18,w-50,43,23,C.navy,true);txt(s,body,x+25,y+67,w-50,h-80,19,C.muted);}

let s=base(1,'Бизнес.Пульс','Учебный проект по управлению программными проектами');
txt(s,title,64,190,620,150,41,C.navy,true);
txt(s,'От выписки к понятной картине бизнеса: динамика, прогноз и советы.',64,365,585,100,24,C.muted);
shape(s,64,502,465,68,C.blue,16);txt(s,'Команда из трёх студентов · 150 часов',83,514,430,44,20,C.white,true);
await photo(s,'screen_dashboard_desktop.png',695,180,520,370);
s.speakerNotes.textFrame.setText('Источник: локальный проект Бизнес.Пульс и скриншот приложения от 07.10.2026.');

s=base(2,'Задача и границы продукта','Для владельца малого бизнеса');
card(s,64,184,354,342,'Проблема','Операции и расходы приходится сводить вручную. Падение поступлений и риск кассового разрыва замечают поздно.',C.orange);
card(s,463,184,354,342,'Решение','Загрузить CSV, увидеть метрики и динамику, получить прогноз на 14 дней и рекомендации.',C.blue);
card(s,862,184,354,342,'Граница MVP','Номер счёта хранится как метаданные. Для прямого банковского API нужны авторизация и адаптер конкретного банка.',C.green);
txt(s,'Система работает с данными счёта любого банка через единый CSV-формат.',65,558,1100,58,23,C.navy,true);

s=base(3,'Как работает система','Путь данных от файла к решению');
const flow=[['1 · Данные','Личный CSV / админский импорт'],['2 · Сервер','Go API · валидация · PostgreSQL'],['3 · Аналитика','Метрики · индекс · прогноз'],['4 · Интерфейс','Дашборд · советы · отчёт']];
flow.forEach((a,i)=>{const x=65+i*304;card(s,x,209,278,260,a[0],a[1],[C.blue,C.green,C.orange,C.blue][i]);if(i<3)txt(s,'→',x+280,300,30,40,32,C.muted,true);});
txt(s,'Дополнительно: Python/Prophet с резервным Go-прогнозом, Telegram-уведомления и опциональный LLM API.',65,520,1120,78,22,C.muted);

s=base(4,'Данные и ежедневный обзор','Скриншоты работающего локального стенда');
await photo(s,'screen_bank_desktop.png',58,179,562,407);
await photo(s,'screen_dashboard_desktop.png',653,179,562,407);
txt(s,'Загрузка операций и данные счёта',62,594,530,35,20,C.navy,true);
txt(s,'Состояние бизнеса и прогноз',657,594,530,35,20,C.navy,true);
s.speakerNotes.textFrame.setText('Оба снимка получены с локального стенда 07.10.2026.');

s=base(5,'Аналитика и рекомендации','Расчёты отделены от генеративного текста');
await photo(s,'screen_analytics_desktop.png',56,182,563,385);
await photo(s,'screen_advice_desktop.png',657,182,562,385);
txt(s,'Графики, операции, экспорт',63,580,520,38,20,C.navy,true);
txt(s,'Правила; AI при настроенном API',660,580,540,38,20,C.navy,true);
s.speakerNotes.textFrame.setText('AI-советы требуют проверки человеком; снимки сделаны 07.10.2026.');

s=base(6,'Команда и календарный план','Роли равноценны по общей нагрузке');
card(s,64,184,354,296,'Студент 1 · 50 ч','Backend: данные, миграции, CSV, личный импорт. QA импорта и API — 8 ч.',C.blue);
card(s,463,184,354,296,'Студент 2 · 50 ч','Backend: метрики, прогноз, LLM и Telegram. QA расчётов и API — 8 ч.',C.green);
card(s,862,184,354,296,'Студент 3 · 50 ч','Frontend: экраны, навигация, UI QA, инструкция и презентация.',C.orange);
txt(s,'01–04.09  требования   →   07–18.09  реализация   →   21–25.09  интеграция   →   28–30.09  QA и сдача',65,540,1140,84,22,C.navy,true);

s=base(7,'Что готово и что проверено','Результат учебного MVP');
card(s,64,180,550,385,'Доступно в проекте','Вход, импорт CSV, расходы, аналитика, индекс состояния, прогноз, советы, Telegram-бот, опциональный LLM-клиент, тестовые сценарии.',C.blue);
card(s,660,180,555,385,'Ограничения и развитие','Прямой API банка не подключён. Точность прогноза на реальных данных не валидирована. Следующий шаг — авторизованный адаптер банка и пользовательская проверка.',C.green);
txt(s,'Проверка: Docker Compose, сборка веб-приложения и автоматические тесты серверной части.',65,584,1140,54,20,C.muted);

const candidatePath=path.join(stagingDir,'candidate.pptx');
await (await PresentationFile.exportPptx(ppt)).save(candidatePath);
const result=await finalizePresentation({workspaceDir,candidatePath,finalPath,pythonExecutable:'/Users/avtoshin/.cache/codex-runtimes/codex-primary-runtime/dependencies/python/bin/python3',integrityValidatorPath:path.join(skillDir,'container_tools/inspect_presentation_package_integrity.py'),layoutValidatorPath:path.join(skillDir,'container_tools/inspect_presentation_layout_geometry.py'),layoutArgs:['--expected-slide-size-emu','12192000,6858000','--validate-heading-fit'],requiredNativeTableOwnerSlides:[],fontPolicy:{basis:'design',families:[font]},verifyArtifactToolImport:true,receiptPath:path.join(stagingDir,'validation.json')});
console.log(JSON.stringify(result));
