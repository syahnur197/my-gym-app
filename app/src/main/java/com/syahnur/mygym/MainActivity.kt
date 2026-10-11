package com.syahnur.mygym

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import org.json.JSONArray
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL
import java.time.LocalDate
import java.time.YearMonth
import java.time.format.DateTimeFormatter
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.coroutines.launch

data class Lift(val name:String="", val sets:Int=3,val reps:Int=10,val kg:Double=0.0)
data class Plan(val id:Long=0,val name:String="",val lifts:List<Lift> = emptyList())
data class Log(val id:Long,val date:String,val workoutName:String,val lifts:List<Lift>)
fun JSONObject.lifts(key:String):List<Lift> {
 val a=optJSONArray(key)?:JSONArray()
 return (0 until a.length()).map { val x=a.getJSONObject(it); Lift(x.optString("name"),x.optInt("sets"),x.optInt("reps"),x.optDouble("weightKg")) }
}
fun liftsJson(items:List<Lift>)=JSONArray().also { a->items.forEach { e->a.put(JSONObject().put("name",e.name).put("sets",e.sets).put("reps",e.reps).put("weightKg",e.kg)) } }
class Api(var base:String,var token:String="") {
 suspend fun call(method:String,path:String,body:JSONObject?=null):String=withContext(Dispatchers.IO) {
  val c=URL(base.trimEnd('/')+path).openConnection() as HttpURLConnection
  c.requestMethod=method;c.connectTimeout=10000;c.readTimeout=10000
  if(token.isNotEmpty())c.setRequestProperty("Authorization","Bearer $token")
  if(body!=null){c.doOutput=true;c.setRequestProperty("Content-Type","application/json");c.outputStream.use{it.write(body.toString().toByteArray())}}
  try {
   val text=(if(c.responseCode in 200..299)c.inputStream else c.errorStream)?.bufferedReader()?.readText().orEmpty()
   if(c.responseCode !in 200..299)throw Exception("HTTP ${c.responseCode}: $text")
   text
  }finally{c.disconnect()}
 }
 suspend fun plans():List<Plan>{val a=JSONArray(call("GET","/api/workouts"));return (0 until a.length()).map{val o=a.getJSONObject(it);Plan(o.getLong("id"),o.getString("name"),o.lifts("exercises"))}}
 suspend fun logs(month:String):List<Log>{val a=JSONArray(call("GET","/api/sessions?month=$month"));return (0 until a.length()).map{val o=a.getJSONObject(it);Log(o.getLong("id"),o.getString("date"),o.getString("workoutName"),o.lifts("exercises"))}}
}
class MainActivity:ComponentActivity(){
 override fun onCreate(savedInstanceState:Bundle?){super.onCreate(savedInstanceState);setContent{MaterialTheme{GymApp()}}}
}
@Composable fun GymApp(){
 val scope= rememberCoroutineScope()
 var url by remember { mutableStateOf("http://10.0.2.2:8080") }
 var username by remember { mutableStateOf("admin") }
 var password by remember { mutableStateOf("") }
 var api by remember { mutableStateOf<Api?>(null) }
 var plans by remember { mutableStateOf<List<Plan>>(emptyList()) }
 var logs by remember { mutableStateOf<List<Log>>(emptyList()) }
 var month by remember { mutableStateOf(YearMonth.now()) }
 var tab by remember { mutableStateOf(0) }
 var message by remember { mutableStateOf("") }
 var editing by remember { mutableStateOf<Plan?>(null) }
 var selectedDate by remember { mutableStateOf(LocalDate.now().toString()) }
 fun refresh(){scope.launch{try{api?.let{plans=it.plans();logs=it.logs(month.toString())}}catch(e:Exception){message=e.message.orEmpty()}}}
 Column(Modifier.fillMaxSize().padding(16.dp),verticalArrangement=Arrangement.spacedBy(10.dp)){
  Text("My Gym",style=MaterialTheme.typography.headlineMedium)
  if(api==null){
   OutlinedTextField(url,{url=it},label={Text("Server URL")},modifier=Modifier.fillMaxWidth())
   OutlinedTextField(username,{username=it},label={Text("Username")},modifier=Modifier.fillMaxWidth())
   OutlinedTextField(password,{password=it},label={Text("Password")},visualTransformation=PasswordVisualTransformation(),modifier=Modifier.fillMaxWidth())
   Button(onClick={scope.launch{try{val a=Api(url);a.token=JSONObject(a.call("POST","/api/login",JSONObject().put("username",username).put("password",password))).getString("token");api=a;password="";refresh()}catch(e:Exception){message=e.message.orEmpty()}}}){Text("Sign in")}
  }else{
   Row(horizontalArrangement=Arrangement.spacedBy(8.dp)){Button(onClick={tab=0;editing=null}){Text("Workouts")};Button(onClick={tab=1;refresh()}){Text("Calendar")};TextButton(onClick={api=null;plans=emptyList()}){Text("Logout")}}
   if(tab==0){
    if(editing==null){
     Button(onClick={editing=Plan(name="New workout")}){Text("Add workout")}
     LazyColumn(verticalArrangement=Arrangement.spacedBy(8.dp)){
      items(plans){p->Card(Modifier.fillMaxWidth()){Column(Modifier.padding(12.dp)){Text(p.name,style=MaterialTheme.typography.titleMedium);Text("${p.lifts.size} exercises")
       Row{TextButton(onClick={editing=p}){Text("Edit")};TextButton(onClick={scope.launch{try{api?.call("POST","/api/sessions",JSONObject().put("workoutId",p.id).put("date",LocalDate.now().toString())) ;message="Session saved";refresh()}catch(e:Exception){message=e.message.orEmpty()}}}){Text("Log today")};TextButton(onClick={scope.launch{try{api?.call("DELETE","/api/workouts/${p.id}");refresh()}catch(e:Exception){message=e.message.orEmpty()}}}){Text("Delete")}}
      }}}}
    }else{
     val original=editing!!
     key(original.id){
      var name by remember {mutableStateOf(original.name)}
      val exercises= remember { mutableStateListOf<Lift>().also{it.addAll(original.lifts)} }
      Text("Edit workout",style=MaterialTheme.typography.titleLarge)
      OutlinedTextField(name,{name=it},label={Text("Workout name")},modifier=Modifier.fillMaxWidth())
      LazyColumn(Modifier.weight(1f),verticalArrangement=Arrangement.spacedBy(8.dp)){
       items(exercises.size){i->val lift=exercises[i]
        Card{Column(Modifier.padding(8.dp)){
         OutlinedTextField(lift.name,{exercises[i]=lift.copy(name=it)},label={Text("Exercise")})
         Row(horizontalArrangement=Arrangement.spacedBy(4.dp)){
          OutlinedTextField(lift.sets.toString(),{exercises[i]=lift.copy(sets=it.toIntOrNull()?:0)},label={Text("Sets")},modifier=Modifier.weight(1f),keyboardOptions=KeyboardOptions(keyboardType=KeyboardType.Number))
          OutlinedTextField(lift.reps.toString(),{exercises[i]=lift.copy(reps=it.toIntOrNull()?:0)},label={Text("Reps")},modifier=Modifier.weight(1f),keyboardOptions=KeyboardOptions(keyboardType=KeyboardType.Number))
          OutlinedTextField(lift.kg.toString(),{exercises[i]=lift.copy(kg=it.toDoubleOrNull()?:0.0)},label={Text("kg")},modifier=Modifier.weight(1f),keyboardOptions=KeyboardOptions(keyboardType=KeyboardType.Decimal))
         }
         TextButton(onClick={exercises.removeAt(i)}){Text("Remove exercise")}
        }}
       }
      }
      Row{Button(onClick={exercises.add(Lift(name="New exercise"))}){Text("Add exercise")};Spacer(Modifier.width(8.dp));Button(onClick={scope.launch{try{val path=if(original.id==0L)"/api/workouts" else "/api/workouts/${original.id}";api?.call(if(original.id==0L)"POST" else "PUT",path,JSONObject().put("name",name).put("exercises",liftsJson(exercises)));editing=null;refresh()}catch(e:Exception){message=e.message.orEmpty()}}}){Text("Save")};TextButton(onClick={editing=null}){Text("Cancel")}}
     }
    }
   }else{
    Row{TextButton(onClick={month=month.minusMonths(1);refresh()}){Text("‹")};Text(month.format(DateTimeFormatter.ofPattern("MMMM yyyy")),modifier=Modifier.padding(12.dp));TextButton(onClick={month=month.plusMonths(1);refresh()}){Text("›")}}
    val first=month.atDay(1);val offset=first.dayOfWeek.value-1
    for(week in 0..5){
     Row(Modifier.fillMaxWidth()){
      for(col in 0..6){val day=week*7+col-offset+1
       Box(Modifier.weight(1f)){if(day in 1..month.lengthOfMonth()){
        val date=month.atDay(day).toString();val count=logs.count{it.date==date}
        TextButton(onClick={selectedDate=date},contentPadding=PaddingValues(0.dp)){Column{Text(day.toString());if(count>0)Text("●",color=MaterialTheme.colorScheme.primary)}}
       }}
      }
     }
    }
    Text("Sessions: $selectedDate",style=MaterialTheme.typography.titleMedium)
    LazyColumn{items(logs.filter{it.date==selectedDate}){log->Card(Modifier.fillMaxWidth().padding(vertical=4.dp)){Column(Modifier.padding(12.dp)){Text(log.workoutName,style=MaterialTheme.typography.titleMedium);log.lifts.forEach{Text("${it.name}: ${it.sets} × ${it.reps} @ ${it.kg} kg")}}}}}
   }
  }
  if(message.isNotEmpty())Text(message,color=MaterialTheme.colorScheme.error)
 }
}
