package whisk.common

import scala.collection.mutable.ArrayBuffer
//import scala.util.Try
import spray.json._

//import whisk.core.entity.ActivationId
//import whisk.core.entity.WhiskActivation

/**
  * LatencyStack: array of tuples (LogMarkerToken, component-to-component latency)
  * 
  */
case class LatencyStack(val stack: ArrayBuffer[(String,String,Long)] = new ArrayBuffer[(String,String,Long)])

object LatencyStack extends DefaultJsonProtocol {
  //override implicit val serdes = jsonFormat1(LatencyStack.apply _)

  implicit val serdes = new RootJsonFormat[LatencyStack] {
    def write(t: LatencyStack) = JsArray(t.stack.toVector.map(x => JsArray(JsString(x._1),JsString(x._2),JsNumber(x._3))))

    def read(value: JsValue) =
      value match {
        case JsArray(elements) =>
          LatencyStack(elements.map(x =>
            x match {
              case JsArray(Vector(JsString(component), JsString(action), JsNumber(delta))) =>
                ((component, action, delta.longValue))
            }).to[ArrayBuffer])
        case _ => throw DeserializationException("Invalid LatencyStack format: " + value)
      }
  }
}

/**
  * 
  * 
  */
/*case class WhiskActivationResponse(val latencyStack: LatencyStack, val res: Either[ActivationId, WhiskActivation])

object WhiskActivationResponse extends DefaultJsonProtocol {
  override implicit val serdes = jsonFormat2(WhiskActivationResponse.apply)
}*/
