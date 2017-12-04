package whisk.common

import scala.collection.mutable.ArrayBuffer
import scala.util.control.Breaks._
//import scala.util.Try
import spray.json._

//import whisk.core.entity.ActivationId
//import whisk.core.entity.WhiskActivation

/**
  * LatencyStack: array of tuples (LogMarkerToken, component-to-component latency)
  * 
  */
case class LatencyStack(val stack: ArrayBuffer[(String,String,Long)] = new ArrayBuffer[(String,String,Long)]) {
  def amend(other: LatencyStack) = {
    stack ++= other.stack
  }

  def add(hop: (String,String,Long))(implicit logging: Logging) = {
    if (stack.length > 0 && stack.last._1 == "invoker") {
      logging.info(this, "add after invoke")
      var sum: Long = 0
      breakable { for (phop <- stack.reverseIterator) {
        if (phop._1 == hop._1) { // same component
          stack += ((hop._1, hop._2, hop._3 - sum))
          break
        } else {
          sum += phop._3
        }
      } }
    } else {
      logging.info(this, "add before invoke")
      stack += hop
    }
  }
}

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
  * The invoker's result for an activation might only be an ActivationId,
  * e.g. in the case of RecordTooLargeException; see InvokerReactive
  *
  */
/*case class WhiskActivationOutcome(res: Either[ActivationId, WhiskActivation], latencyStack: LatencyStack) {
  override def serialize: String = {
    WhiskActivationOutcome.serdes.write(this).compactPrint
  }
}
object WhiskActivationOutcome extends DefaultJsonProtocol {
  //def parse(msg: String): Try[WhiskActivationOutcome] = Try(serdes.read(msg.parseJson))
  //private val serdes = jsonFormat2(WhiskActivationOutcome.apply)
  def parse(msg: String) = Try(serdes.read(msg.parseJson))
  implicit val serdes = jsonFormat2(WhiskActivationOutcome.apply)
}
 */
